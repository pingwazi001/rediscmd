package util

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/garyburd/redigo/redis"
)

//redis连接池
var redisPool *redis.Pool

var DbCount int //数据库数量

//redis缓存key信息
var (
	CacheKeysMap sync.Map
)

//获取连接的锁对象
var CreateConnectLock sync.Mutex

//数据库信息
type DBInfo struct {
	dbid, dbKeys int
}

//初始化连接池信息
func init() {
	for {
		if err := initRedisConnectInfo(); err != nil {
			log.Println(err)
			continue
		}
		break
	}
}

//初始化redis连接信息
func initRedisConnectInfo() error {
	conf, err := GetConf()
	if err != nil {
		return err
	}
	redisPool = &redis.Pool{
		Dial: func() (conn redis.Conn, e error) {
			c, err := redis.Dial("tcp", fmt.Sprintf("%s:%d", conf.Redis.AddRess, conf.Redis.Port))
			c.Do("AUTH", conf.Redis.Password) //认证
			return c, err
		},
		MaxIdle:     1,                     //连接池中最小空闲连接数
		MaxActive:   conf.Redis.MaxConnect, //线程此的最大连接数
		IdleTimeout: 60 * time.Second,      //30s之后，关闭多余的空闲连接
	}
	if err := initDBCount(); err != nil {
		return err
	} //初始化数据库数量
	initLoadAllCacheKeys() //触发加载所有数据库中的缓存key信息
	return nil
}

//获取连接要加锁
func createConnection() redis.Conn {
	CreateConnectLock.Lock()
	for redisPool.ActiveCount() == redisPool.MaxActive {
		time.Sleep(5 * time.Millisecond) //休眠一定时间，等待连接池空闲
	}
	conn := redisPool.Get()
	CreateConnectLock.Unlock()
	return conn
}

//加载所有数据库的缓存key到本地
func initLoadAllCacheKeys() {
	for dbid := 0; dbid < DbCount; dbid++ {
		go func(dbIdItem int) {
			conn := createConnection()
			defer conn.Close()
			conn.Do("select", dbIdItem)
			keysRet, err := conn.Do("keys", "*")
			if err != nil {
				log.Println(err)
			}
			keysMap := make(map[string][]string)
			if keysRet == nil { //当前数据库没有缓存key
				CacheKeysMap.Store(dbIdItem, keysMap)
				return
			}

			for _, item := range keysRet.([]interface{}) {
				key := string(item.([]uint8))
				if _, ok := keysMap[key]; !ok {
					keysMap[strings.ToLower(key)] = []string{key}
				} else {
					keysMap[strings.ToLower(key)] = append(keysMap[strings.ToLower(key)], key)
				}
			}
			CacheKeysMap.Store(dbIdItem, keysMap)
		}(dbid)
	}
}

//获取数据库的数量
func initDBCount() error {
	connection := createConnection()
	defer createConnection().Close()
	connection.Do("select", "0") //选择指定数据库
	ret, err := connection.Do("config", "get", "databases")
	if err != nil {
		return err
	}
	ret1, ok := ret.([]interface{})
	if !ok {
		return fmt.Errorf("获取数据库数量信息时发生类型转换错误")
	}
	DbCount, _ = strconv.Atoi(string(ret1[1].([]uint8)))
	return nil
}

//获取指定数据库中的keys数量
func loadDBKeysCount(dbInfoChan chan DBInfo, dbid int) {
	c := createConnection()
	if c == nil {
		log.Println("获取连接失败")
	}
	defer c.Close()                    //关闭连接
	c.Do("select", strconv.Itoa(dbid)) //选择指定数据库
	dbSize, err := c.Do("dbsize")      //获取尺寸
	if err != nil {
		dbInfoChan <- DBInfo{dbid: dbid, dbKeys: 0}
		return
	}
	dbInfoChan <- DBInfo{dbid: dbid, dbKeys: int(dbSize.(int64))}
}

//加载数据库列表
func LoadAllDBs(isAll bool, count int) map[int]int {
	dbCount := DbCount //获取数据库的数量
	if !isAll {
		dbCount = count
	}
	dbInfoChan := make(chan DBInfo, dbCount)
	for i := 0; i < dbCount; i++ {
		go loadDBKeysCount(dbInfoChan, i)
	}
	dbMaps := make(map[int]int, dbCount)
	putCount := 1
	var dbSize DBInfo
	for i := 0; i < dbCount; i++ {
		if putCount <= dbCount {
			putCount++
			dbSize = <-dbInfoChan
			dbMaps[dbSize.dbid] = dbSize.dbKeys
		}
		if _, exists := dbMaps[i]; !exists {
			i--
			time.Sleep(5 * time.Millisecond) //休眠5毫秒，避免cpu空转
			continue
		}
	}
	close(dbInfoChan)
	return dbMaps
}

//加载指定数据库的所有缓存key
func SearchKeys(dbid int, pattern string) ([]string, error) {
	if pattern == "" {
		pattern = "*"
	}
	var keysMap map[string][]string
	firstPrint := true
	for {
		if ret, ok := CacheKeysMap.Load(dbid); ok {
			keysMap = ret.(map[string][]string)
			break //缓存key已加载完成，可以继续后续操作
		}
		if !firstPrint {
			log.Print(".")
		} else {
			firstPrint = false
			log.Println("后台正在加载当前数据库中的缓存Key，请...")
		}
		time.Sleep(1 * time.Second) //休眠一秒
	}

	pattern = strings.ToLower(pattern)
	pattern = strings.ReplaceAll(pattern, ".", "\\.")
	pattern = strings.ReplaceAll(pattern, "*", ".*")
	retKeys := make([]string, 0)
	for k := range keysMap {
		if ok, _ := regexp.Match(pattern, []byte(k)); ok {
			retKeys = append(retKeys, keysMap[k]...)
		}
	}
	return retKeys, nil
}

//获取值
func GetValue(key string, dbid int) (string, error) {
	if key == "" {
		return "", errors.New("key不能为空")
	}
	conn := createConnection()
	defer conn.Close()
	conn.Do("select", dbid)
	ret, err := conn.Do("get", key)
	if err != nil {
		return "", errors.New("获取值错误，")
	}
	if ret == nil {
		return "", errors.New("为查询到任何值")
	}
	if _, ok := ret.([]uint8); !ok {
		return "", errors.New("值解析错误")
	}
	return string(ret.([]uint8)), nil
}

//设置值
func SetValue(dbid int, key, value string) {
	if key == "" {
		return
	}
	conn := createConnection()
	defer conn.Close()
	conn.Do("select", dbid)
	conn.Do("set", key, value)
}

//删除缓存key
func DeleteKey(dbid int, key ...string) {
	if len(key) == 0 {
		return
	}
	conn := createConnection()
	defer conn.Close()
	conn.Do("select", dbid)
	conn.Do("del", key)
}

func FlushDB(dbid int) {
	conn := createConnection()
	defer conn.Close()
	conn.Do("select", dbid)
	conn.Do("FLUSHDB")
}
