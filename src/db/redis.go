package db

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"rediscmd/src/conf"

	"rediscmd/src/model"

	"github.com/garyburd/redigo/redis"
)

var (
	redisPool              *redis.Pool //redis连接池
	redisDBCount           int         //数据库数量
	redisOptionDBId        = 0         //操作的redis数据库id
	createRedisConnectLock sync.Mutex  //获取redis连接的锁对象
)

//初始化redis连接池
func initRedisPool() error {
	conf, err := conf.GetRedisConf()
	if err != nil {
		return err
	}
	redisPool = &redis.Pool{
		Dial: func() (conn redis.Conn, e error) {
			c, err := redis.Dial("tcp", fmt.Sprintf("%s:%d", conf.Redis.AddRess, conf.Redis.Port), redis.DialConnectTimeout(30*time.Second))
			if err != nil {
				return nil, err
			}
			if _, err := c.Do("AUTH", conf.Redis.Password); err != nil {
				return nil, err
			} //认证
			return c, nil
		},
		MaxIdle:     1,                     //连接池中最小空闲连接数
		MaxActive:   conf.Redis.MaxConnect, //线程此的最大连接数
		IdleTimeout: 60 * time.Second,      //30s之后，关闭多余的空闲连接
	}
	return nil
}

//获取连接要加锁
func createRedisConnection() (redis.Conn, error) {
	createRedisConnectLock.Lock()
	for redisPool.ActiveCount() == redisPool.MaxActive {
		time.Sleep(5 * time.Millisecond) //休眠一定时间，等待连接池空闲
	}
	conn := redisPool.Get()
	createRedisConnectLock.Unlock() //释放锁
	if err := conn.Err(); err != nil {
		return nil, err
	}
	conn.Do("select", redisOptionDBId) //指定操作的数据库id
	return conn, nil
}

//初始化redis信息
func InitRedisInfo(isSelectConfName bool) {
	forCount := 3
	for forCount >= 0 {
		forCount--
		if isSelectConfName {
			conf.SeleRedisctConfFileName() //选择配置文件名称
		}
		err := conf.CheckRedisConf() //检查配置文件内容
		if err != nil {
			log.Printf("配置文件检查报错%s，请重新填写此配置文件内容！", err.Error())
			initErr := conf.InitRedisConf() //配置文件检查不通过就重新初始化此配置文件内容
			if initErr != nil {
				log.Println(initErr)
			}
			continue
		}
		//初始化redis连接
		if err := initRedisPool(); err != nil {
			log.Printf("redis连接池初始化报错%s，重新初始化！", err.Error())
			continue
		}
		if err := initRedisDBCount(); err != nil {
			log.Printf("初始化获取redis的数据库数量报错%s，重新初始化！", err.Error())
			continue
		} //初始化数据库数量
		break
	}
}

//获取数据库的数量
func initRedisDBCount() error {
	connection, err := createRedisConnection()
	if err != nil {
		return err
	}
	defer connection.Close()
	connection.Do("select", "0") //选择指定数据库
	ret, err := connection.Do("config", "get", "databases")
	if err != nil {
		return err
	}
	ret1, ok := ret.([]interface{})
	if !ok {
		return fmt.Errorf("获取数据库数量信息时发生类型转换错误")
	}
	redisDBCount, _ = strconv.Atoi(string(ret1[1].([]uint8)))
	return nil
}

//reids的数据库信息
func AllRedisDBInfo(isAll bool, count int, dbInfoChan chan model.RedisDBInfo) {
	defer close(dbInfoChan) //关闭通道
	dbCount := redisDBCount //获取数据库的数量
	if !isAll {
		dbCount = count
	}
	var wg sync.WaitGroup
	for i := 0; i < dbCount; i++ {
		wg.Add(1)
		go func(dbid int, waitG *sync.WaitGroup) {
			defer waitG.Done()
			connection, err := createRedisConnection()
			if err != nil {
				log.Println(err)
				dbInfoChan <- model.RedisDBInfo{DBId: dbid, DBKeys: 0}
				return
			}
			defer connection.Close()
			connection.Do("select", dbid) //选择指定数据库
			ret, err := connection.Do("dbsize")
			if err != nil {
				log.Println(err)
				dbInfoChan <- model.RedisDBInfo{DBId: dbid, DBKeys: 0}
				return
			}
			keysCount, ok := ret.(int64)
			if !ok {
				log.Println("读取数据库缓存key的数量转换失败")
				dbInfoChan <- model.RedisDBInfo{DBId: dbid, DBKeys: 0}
				return
			}
			dbInfoChan <- model.RedisDBInfo{DBId: dbid, DBKeys: keysCount}
		}(i, &wg)
	}
	wg.Wait()
}

func SearchRedisKeysIgnoreCase(pattern string, keysChan chan string) {
	defer close(keysChan) //关闭通道
	conf, err := conf.GetRedisConf()
	if err != nil {
		log.Println(err)
		return
	}
	pattern = strings.ToLower(pattern)
	pattern = strings.ReplaceAll(pattern, ".", "\\.")
	pattern = strings.ReplaceAll(pattern, "*", ".*")

	keyPrefixs := strings.Split(conf.Redis.KeyPrefix, ",")
	var wg sync.WaitGroup
	wg.Add(len(keyPrefixs))
	for _, prefixItem := range keyPrefixs {
		go func(prefix string, waitG *sync.WaitGroup) {
			defer waitG.Done() //标记任务已结束
			conn, err := createRedisConnection()
			if err != nil {
				log.Println(err)
				return
			}
			defer conn.Close()
			keysRet, err := conn.Do("keys", fmt.Sprintf("%s*", prefix))
			if err != nil {
				log.Println(err)
				return
			}
			if keysRet == nil { //当前数据库没有缓存key
				return
			}
			matchPatternKeys(pattern, keysRet.([]interface{}), keysChan)
		}(prefixItem, &wg)
	}
	wg.Wait() //等待结束，释放通道资源
}

func matchPatternKeys(pattern string, keys []interface{}, keysChan chan string) {
	if len(keys) <= 0 {
		return
	}
	for _, item := range keys {
		key := string(item.([]uint8))
		key = strings.ReplaceAll(key, " ", "")
		if key == "" {
			continue
		}
		regKey := strings.ToLower(key)
		if ok, _ := regexp.Match(pattern, []byte(regKey)); ok {
			keysChan <- key
			continue
		}
	}
}

//模糊查询缓存key
func SearchRedisKeys(pattern string) []string {
	if pattern == "" {
		pattern = "*"
	}
	retKeys := []string{}
	conn, err := createRedisConnection()
	if err != nil {
		log.Println(err)
		return retKeys
	}
	keysRet, err := conn.Do("keys", pattern)
	if err != nil {
		log.Println(err)
		return retKeys
	}
	if keysRet == nil { //当前数据库没有缓存key
		return retKeys
	}
	for _, item := range keysRet.([]interface{}) {
		key := string(item.([]uint8))
		key = strings.ReplaceAll(key, " ", "")
		if key == "" {
			continue
		}
		retKeys = append(retKeys, key)
	}
	return retKeys
}

//获取指定key的值
func GetRedisValue(key string) (string, error) {
	if key == "" {
		return "", errors.New("key不能为空")
	}
	conn, err := createRedisConnection()
	if err != nil {
		return "", err
	}
	defer conn.Close()
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

//给指定key设置值
func SetRedisValue(key, value string) {
	if key == "" {
		return
	}
	conn, err := createRedisConnection()
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()
	conn.Do("set", key, value)
}

//删除key的缓存
func DeleteRedisKey(key ...string) {
	if len(key) == 0 {
		return
	}
	conn, connErr := createRedisConnection()
	if connErr != nil {
		log.Println(connErr)
		return
	}
	defer conn.Close()
	keys := make([]interface{}, 0, len(key))
	for _, item := range key {
		keys = append(keys, item)
	}
	_, err := conn.Do("del", keys...)
	if err != nil {
		log.Println(err)
	}
}

//清空数据库中的所有缓存
func FlushRedisDB() {
	conn, connErr := createRedisConnection()
	if connErr != nil {
		log.Println(connErr)
		return
	}
	defer conn.Close()
	conn.Do("FLUSHDB")
}

//切换redis的操作数据库
func ChangeRedisOptionDBId(dbid int) {
	if dbid < 0 || dbid >= redisDBCount {
		log.Printf("数据库切换失败，请输入[0~%d)的数据库编号！", redisDBCount)
		return
	}
	if dbid == redisOptionDBId {
		return
	}
	//切换操作的数据库id
	redisOptionDBId = dbid
	for {
		if err := initRedisPool(); err != nil {
			log.Println(err)
			continue
		}
		break
	}
}

//redis的数据库数量
func RedisDBCount() int {
	return redisDBCount
}

//当前操作的数据id
func RedisOptionDBId() int {
	return redisOptionDBId
}
