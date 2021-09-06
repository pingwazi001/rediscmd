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
	redisCacheKeysMap      sync.Map    //redis缓存key信息
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
		TriggerLoadAllCacheKeysToLocal() //触发加载所有数据库中的缓存key信息
		break
	}
}

//触发加载所有缓存key到本地
func TriggerLoadAllCacheKeysToLocal() {
	//遍历删除已加载的缓存key
	redisCacheKeysMap.Range(func(k, v interface{}) bool {
		redisCacheKeysMap.Delete(k)
		return true
	})

	for dbid := 0; dbid < redisDBCount; dbid++ {
		go func(dbIdItem int) {
			conn, err := createRedisConnection()
			if err != nil {
				log.Println(err)
				return
			}
			defer conn.Close()
			conn.Do("select", dbIdItem)
			keysRet, err := conn.Do("keys", "*")
			if err != nil {
				log.Println(err)
			}
			keysMap := make(map[string][]string)
			if keysRet == nil { //当前数据库没有缓存key
				redisCacheKeysMap.Store(dbIdItem, keysMap)
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
			redisCacheKeysMap.Store(dbIdItem, keysMap)
		}(dbid)
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

//获取指定数据库中的keys数量
func loadDBKeysCount(dbInfoChan chan model.RedisDBInfo, dbid int) {
	isPrinted := false
	for {
		if ret, ok := redisCacheKeysMap.Load(dbid); ok {
			keysMap := ret.(map[string][]string)
			dbInfoChan <- model.RedisDBInfo{DBId: dbid, DBKeys: len(keysMap)}
			break
		}
		if !isPrinted {
			log.Printf("后台正在加载数据库编号=%d中的所有缓存Key到本地，请稍后...\r\n", dbid)
			isPrinted = true
		}
		time.Sleep(1 * time.Second) //休眠一秒
	}
}

//reids的数据库信息
func AllRedisDBInfo(isAll bool, count int) chan model.RedisDBInfo {
	dbCount := redisDBCount //获取数据库的数量
	if !isAll {
		dbCount = count
	}
	dbInfoChan := make(chan model.RedisDBInfo, dbCount)
	for i := 0; i < dbCount; i++ {
		go loadDBKeysCount(dbInfoChan, i)
	}
	return dbInfoChan
}

func LoadRedisKeys() {
	conf, err := conf.GetRedisConf()
	if err != nil {
		log.Fatal(err)
	}

	keyPrefixs := strings.Split(conf.Redis.KeyPrefix, ",")
	var wg sync.WaitGroup
	wg.Add(len(keyPrefixs))
	for _, prefixItem := range keyPrefixs {
		go func(waitG *sync.WaitGroup) {
			defer waitG.Done()
			conn, err := createRedisConnection()
			if err != nil {
				log.Println(err)
				return
			}
			defer conn.Close()
			keysRet, err := conn.Do("keys", fmt.Sprintf("%s*", prefixItem))
			if err != nil {
				log.Println(err)
			}
			if keysRet == nil { //当前数据库没有缓存key
				return
			}
			for _, itemKey := range keysRet.([]interface{}) {
				key := string(itemKey.([]uint8))
				fmt.Println(key)
			}
		}(&wg)
	}
	wg.Wait()
}

//模糊查询缓存key
func SearchRedisKeys(pattern string) (chan []string, error) {
	if pattern == "" {
		pattern = "*"
	}
	var keysMap map[string][]string
	firstPrint := true
	for {
		if ret, ok := redisCacheKeysMap.Load(redisOptionDBId); ok {
			keysMap = ret.(map[string][]string)
			break //缓存key已加载完成，可以继续后续操作
		}
		if !firstPrint {
			fmt.Print(".")
		} else {
			firstPrint = false
			fmt.Print("后台正在加载当前数据库中的缓存Key，请稍候")
		}
		time.Sleep(1 * time.Second) //休眠一秒
	}

	pattern = strings.ToLower(pattern)
	pattern = strings.ReplaceAll(pattern, ".", "\\.")
	pattern = strings.ReplaceAll(pattern, "*", ".*")
	//retKeys := make([]string, 0)
	matchKeysChan := make(chan []string, len(keysMap))
	for k := range keysMap {
		go func(itemKey string) {
			if ok, _ := regexp.Match(pattern, []byte(itemKey)); ok {
				matchKeysChan <- keysMap[itemKey]
				return
			}
			matchKeysChan <- make([]string, 0)
		}(k)
	}
	// for i := 0; i < len(keysMap); i++ {
	// 	retKeys = append(retKeys, <-matchKeysChan...)
	// }
	return matchKeysChan, nil
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
