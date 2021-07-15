package util

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/garyburd/redigo/redis"
)

//redis连接池
var redisPool *redis.Pool

//数据库信息
type DBInfo struct {
	dbid, dbKeys int
}

//初始化连接池信息
func init() {
	conf, err := GetConf()
	if err != nil {
		fmt.Println("配置文件加载失败")
		return
	}
	redisPool = &redis.Pool{
		Dial: func() (conn redis.Conn, e error) {
			c, err := redis.Dial("tcp", fmt.Sprintf("%s:%d", conf.Redis.AddRess, conf.Redis.Port))
			c.Do("AUTH", conf.Redis.Password) //认证
			return c, err
		},
		MaxIdle:     3,
		MaxActive:   100,
		IdleTimeout: 240 * time.Second,
	}
}

//获取一个连接
func getConnection() redis.Conn {
	for redisPool.ActiveCount() == redisPool.MaxActive {
		time.Sleep(5 * time.Millisecond) //休眠一定时间，等待连接池空闲
	}
	return redisPool.Get()
}

//获取数据库的数量
func DBCounts() int {
	connection := getConnection()
	defer getConnection().Close()
	connection.Do("select", "0") //选择指定数据库
	ret, err := connection.Do("config", "get", "databases")
	if err != nil {
		log.Fatal("数据库列表加载错误，", err)
	}
	ret1, ok := ret.([]interface{})
	if !ok {
		log.Fatal("类型转换失败")
	}
	dbCount, _ := strconv.Atoi(string(ret1[1].([]uint8)))
	return dbCount
}

//获取指定数据库中的keys数量
func loadDBKeysCount(dbInfoChan chan DBInfo, dbid int) {
	c := getConnection()
	if c == nil {
		fmt.Println("获取连接失败")
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
	dbCount := DBCounts() //获取数据库的数量
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

	conn := getConnection()
	defer conn.Close()
	conn.Do("select", dbid)
	keysRet, err := conn.Do("keys", "*")
	if err != nil {
		return nil, errors.New("加载指定数据的所有key失败")
	}
	keysMap := make(map[string][]string)
	for _, item := range keysRet.([]interface{}) {
		key := string(item.([]uint8))
		if _, ok := keysMap[key]; !ok {
			keysMap[strings.ToLower(key)] = []string{key}
		} else {
			keysMap[strings.ToLower(key)] = append(keysMap[strings.ToLower(key)], key)
		}
	}

	pattern = strings.ToLower(pattern)
	pattern = strings.ReplaceAll(pattern, ".", "\\.")
	pattern = strings.ReplaceAll(pattern, "*", ".*")
	retKeys := make([]string, 10)
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
	conn := getConnection()
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
	conn := getConnection()
	defer conn.Close()
	conn.Do("select", dbid)
	conn.Do("set", key, value)
}

//删除缓存key
func DeleteKey(dbid int, key string) {
	if key == "" {
		return
	}
	conn := getConnection()
	defer conn.Close()
	conn.Do("select", dbid)
	conn.Do("del", key)
}
