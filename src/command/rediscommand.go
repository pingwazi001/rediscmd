package command

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"rediscmd/src/conf"
	"rediscmd/src/db"
	"rediscmd/src/model"
	"rediscmd/src/util"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/modood/table"
)

var InputReader *bufio.Reader

type cmdParamfunc func([]string)

//启动程序
func RedisCMDStart() {
	defer func() {
		if err := recover(); err != nil {
			log.Println("工具发生致命错误！请通过https://github.com/pwzos/rediscmd向工具作者进行反馈")
			log.Println(err)
			log.Println("按回车退出程序...")
			InputReader.ReadString('\n')
			os.Exit(1)
		}
	}()
	db.InitRedisInfo(true) //初始化redis信息
	funcOptionMsg()        //功能提示语
	for {
		funcOption() //功能选择
	}
}

//功能列表选择
func funcOptionMsg() {
	log.Println("https://github.com/pwzos/rediscmd")
	log.Println(conf.RedisConfAbsPath())
	content, err := util.ReadFileAsString(conf.RedisConfAbsPath())
	if err != nil {
		log.Println(err)
	}
	log.Println(conf.RedisConfName() + "\r\n" + content)
	msg := []model.KV{
		{Key: "cls", Value: "清屏"},
		{Key: "keys", Value: "模糊查询缓存key [y:忽略大小写|不传或n:精确] [keypattern]"},
		{Key: "get", Value: "查询模糊key的值 [y:忽略大小写|不传或n:精确] [keypattern]"},
		{Key: "del", Value: "写删除模糊key的值 [y:忽略大小写|不传或n:精确] [keypattern]"},
		{Key: "set", Value: "设置精确key的值 [key] [value]"},
		{Key: "ldb", Value: fmt.Sprintf("加载数据库列表 [y|n] [0~%d)", db.RedisDBCount())},
		{Key: "resetconf", Value: fmt.Sprintf("重新配置%s文件内容", conf.RedisConfName())},
		{Key: "changeconf", Value: "切换配置文件"},
		{Key: "addconf", Value: "新增配置文件"},
		{Key: "changeoptdbid", Value: fmt.Sprintf("切换当前操作数据库编号%d为其他值 [0~%d)", db.RedisDBCount(), db.RedisDBCount())},
		{Key: "quit", Value: "退出程序"},
	}
	t := table.AsciiTable(msg)
	fmt.Println(t)
}

//功能选择
func funcOption() {
	defer func() {
		if err := recover(); err != nil {
			log.Println("工具发生致命错误！请通过https://github.com/pwzos/rediscmd向工具作者进行反馈")
			log.Println(err)
			log.Println("按任意键继续操作...")
			InputReader.ReadString('\n')
			return
		}
	}()
	option, _ := util.ReadValueFromConsole("请输出操作命令(回车结束输入)", false)
	cmdParams := strings.Split(option, " ")
	cmdParams = dealCMDParams(cmdParams) //处理命令行参数
	switch cmdParams[0] {
	case "cls":
		util.ClearConsoleScreen()
		funcOptionMsg()
	case "keys":
		keysOptionCMD("模糊查询缓存Key的列表需要2~3个参数，请重新输入", cmdParams, keysCMD, keysIgnoreCaseCMD)
	case "get":
		keysOptionCMD("模糊查询缓存Key的值需要2~2个参数，请重新输入", cmdParams, getCMD, getIgnoreCaseCMD)
	case "del":
		keysOptionCMD("模糊批量删除缓存需要2~3个参数，请重新输入", cmdParams, delCMD, delIgnoreCaseCMD)
	case "set":
		setCMD(cmdParams)
	case "ldb":
		loadDBCMD(cmdParams)
	case "resetconf":
		resetConfCMD()
	case "changeconf":
		changeConfCMD()
	case "addconf":
		addConfCMD()
	case "changeoptdbid":
		changeOptDbIdCMD(cmdParams)
	case "quit":
		os.Exit(1)
	default:
		log.Printf("您输入的操作【%s】不支持！请重新输入", cmdParams[0])
	}
}

//处理输入的命令行参数
func dealCMDParams(cmdParams []string) []string {
	retCmdParams := []string{}
	for _, item := range cmdParams {
		item = strings.Trim(item, " ")
		if item == "" {
			continue
		}
		retCmdParams = append(retCmdParams, item)
	}
	return retCmdParams
}

//检查命令行参数个数是否符合规则
func checkCMDParamsCount(cmdParams []string, count int) bool {
	return count <= len(cmdParams)
}

//加载数据库列表信息
func loadDBCMD(cmdParams []string) {
	if !checkCMDParamsCount(cmdParams, 2) {
		log.Println("加载数据库列表信息至少需要两个参数，请重新输入")
		return
	}
	opt := cmdParams[1]
	loadDbCount := db.RedisDBCount()
	isLoadAll := true
	switch opt {
	case "y":
		log.Println("正在加载全部数据库信息，请稍候...")
	case "n":
		if !checkCMDParamsCount(cmdParams, 3) {
			log.Println("请输入需要加载前前多少个数据库的信息")
			return
		}
		count, err := strconv.Atoi(cmdParams[2])
		if err != nil || count <= 0 {
			log.Println("您的输入的数量无法解析，请重来")
			return
		}
		loadDbCount = count
		isLoadAll = false
	default:
		log.Println("加载方式仅允许输入y/n")
		return
	}

	dbInfoChan := make(chan model.RedisDBInfo, loadDbCount)
	go db.AllRedisDBInfo(isLoadAll, loadDbCount, dbInfoChan)

	dbMaps := make(map[int]int64, loadDbCount)
	putCount := 1
	var dbSize model.RedisDBInfo
	for i := 0; i < loadDbCount; i++ {
		if putCount <= loadDbCount {
			putCount++
			dbSize = <-dbInfoChan
			dbMaps[dbSize.DBId] = dbSize.DBKeys
		}
		keysCount, exists := dbMaps[i]
		if !exists {
			i--
			time.Sleep(5 * time.Millisecond) //休眠5毫秒，避免cpu空转
			continue
		} else {
			log.Printf("db(%d)=%d\r\n", i, keysCount)
		}
	}
}

func keysOptionCMD(paramErrMsg string, cmdParams []string, cmdFunc cmdParamfunc, ignoreCaseCmdFunc cmdParamfunc) {
	if !checkCMDParamsCount(cmdParams, 2) && !checkCMDParamsCount(cmdParams, 3) {
		log.Println(paramErrMsg)
		return
	}
	if len(cmdParams) == 3 && cmdParams[1] == "y" {
		ignoreCaseCmdFunc(append(cmdParams[0:1], cmdParams[2]))
	} else if len(cmdParams) == 2 && cmdParams[1] == "n" {
		cmdFunc(append(cmdParams[0:1], cmdParams[2]))
	} else if len(cmdParams) == 2 {
		cmdFunc(cmdParams)
	} else {
		log.Println("参数不符合规则")
	}
}

//加载缓存key
func keysCMD(cmdParams []string) {
	if !checkCMDParamsCount(cmdParams, 2) {
		log.Println("模糊查询缓存Key的列表需要2个参数，请重新输入")
		return
	}
	keys := db.SearchRedisKeys(cmdParams[1])
	for _, key := range keys {
		log.Println(key)
	}

}

//不区分大小写加载缓存key
func keysIgnoreCaseCMD(cmdParams []string) {
	if !checkCMDParamsCount(cmdParams, 2) {
		log.Println("模糊查询缓存Key的列表需要两个参数，请重新输入")
		return
	}
	keysChan := make(chan string, 1000)
	go db.SearchRedisKeysIgnoreCase(cmdParams[1], keysChan) //查询redis缓存key
	for {
		select {
		case key, ok := <-keysChan:
			{
				if !ok {
					log.Println("查询结束")
					return //方法结束
				}
				key = strings.ReplaceAll(key, " ", "")
				if key != "" {
					log.Println(key)
				}
			}
		default:
			{
				fmt.Print(".")
				time.Sleep(1 * time.Second) //休眠一秒
			}
		}
	}
}

//区分大小写的方式获取模糊key的值
func getCMD(cmdParams []string) {
	if !checkCMDParamsCount(cmdParams, 2) {
		log.Println("模糊查询缓存Key的值需要2个参数，请重新输入")
		return
	}
	keys := db.SearchRedisKeys(cmdParams[1])
	var wg sync.WaitGroup
	for _, key := range keys {
		wg.Add(1)
		go func(itemKey string, waitG *sync.WaitGroup) {
			defer waitG.Done()
			value, err := db.GetRedisValue(itemKey)
			if err != nil {
				log.Println(fmt.Sprintf("%s=%s", itemKey, err.Error()))
				return
			}
			log.Println(fmt.Sprintf("%s=%s", itemKey, value))
		}(key, &wg)
	}
	wg.Wait()
}

//不区分大小写获取指定key的值
func getIgnoreCaseCMD(cmdParams []string) {
	if !checkCMDParamsCount(cmdParams, 2) {
		log.Println("模糊查询缓存Key的值需要两个参数，请重新输入")
		return
	}
	keysChan := make(chan string, 1000)
	go db.SearchRedisKeysIgnoreCase(cmdParams[1], keysChan) //查询redis缓存key
	var wg sync.WaitGroup
	for {
		select {
		case key, ok := <-keysChan:
			{
				if !ok {
					wg.Wait()
					log.Println("查询结束")
					return //方法结束
				}
				wg.Add(1)
				go func(itemKey string, waitG *sync.WaitGroup) {
					defer waitG.Done()
					value, err := db.GetRedisValue(itemKey)
					if err != nil {
						log.Println(fmt.Sprintf("%s=%s", itemKey, err.Error()))
						return
					}
					log.Println(fmt.Sprintf("%s=%s", itemKey, value))
				}(key, &wg)
			}
		default:
			{
				fmt.Print(".")
				time.Sleep(1 * time.Second) //休眠一秒
			}
		}
	}
}

//区分大小写的方式模糊删除key的值
func delCMD(cmdParams []string) {
	if !checkCMDParamsCount(cmdParams, 2) || !checkCMDParamsCount(cmdParams, 3) {
		log.Println("模糊批量删除缓存需要2个参数，请重新输入")
		return
	}
	pattern := cmdParams[1]
	if pattern == "*" {
		isSure, _ := util.ReadValueFromConsole(fmt.Sprintf("根据您输入的模糊Key=%s此次操作将清空数据库dbid=%d中的所有缓存！请确认是否执行此操作(y/n):", pattern, db.RedisDBCount()), false)
		if isSure == "y" {
			log.Println("正在处理，请稍候...")
			db.FlushRedisDB() //清空数据库
		}
		return
	}

	keys := db.SearchRedisKeys(cmdParams[1])
	var wg sync.WaitGroup
	delKeysCount := 0
	for _, key := range keys {
		delKeysCount++
		wg.Add(1)
		go func(itemKey string, waitG *sync.WaitGroup) {
			defer waitG.Done()
			start := time.Now()
			db.DeleteRedisKey(itemKey)
			log.Println(fmt.Sprintf("%s 删除成功，耗时%d毫秒", itemKey, time.Since(start).Milliseconds()))
		}(key, &wg)
	}
	wg.Wait()
	log.Printf("共删除%d个缓存", delKeysCount)

}

//不区分大小写删除模糊key的值
func delIgnoreCaseCMD(cmdParams []string) {
	if !checkCMDParamsCount(cmdParams, 2) {
		log.Println("模糊批量删除缓存需要两个参数，请重新输入")
		return
	}
	pattern := cmdParams[1]
	if pattern == "*" {
		isSure, _ := util.ReadValueFromConsole(fmt.Sprintf("根据您输入的模糊Key=%s此次操作将清空数据库dbid=%d中的所有缓存！请确认是否执行此操作(y/n):", pattern, db.RedisDBCount()), false)
		if isSure == "y" {
			log.Println("正在处理，请稍候...")
			db.FlushRedisDB() //清空数据库
		}
		return
	}
	keysChan := make(chan string, 1000)
	go db.SearchRedisKeysIgnoreCase(cmdParams[1], keysChan) //查询redis缓存key
	var wg sync.WaitGroup
	delKeysCount := 0

	for {
		select {
		case key, ok := <-keysChan:
			{
				if !ok {
					wg.Wait()
					log.Printf("共删除%d个缓存", delKeysCount)
					return //方法结束
				}
				delKeysCount++
				wg.Add(1)
				go func(itemKey string, waitG *sync.WaitGroup) {
					defer waitG.Done()
					start := time.Now()
					db.DeleteRedisKey(itemKey)
					log.Println(fmt.Sprintf("%s 删除成功，耗时%d毫秒", itemKey, time.Since(start).Milliseconds()))
				}(key, &wg)
			}
		default:
			{
				fmt.Print(".")
				time.Sleep(1 * time.Second) //休眠一秒
			}
		}
	}
}

func setCMD(cmdParams []string) {
	if !checkCMDParamsCount(cmdParams, 3) {
		log.Println("给指定key设置值需要三个参数，请重新输入")
		return
	}
	key := cmdParams[1]
	value := cmdParams[2]
	db.SetRedisValue(key, value)
}

//重新配置当前设置当前配置文件的内容
func resetConfCMD() {
	initErr := conf.InitRedisConf() //重新配置当前redis连接信息
	if initErr != nil {
		log.Println(initErr)
		return
	}
	checkErr := conf.CheckRedisConf()
	if checkErr != nil {
		log.Println(checkErr)
		return
	}
	db.InitRedisInfo(false) //因为重新配置了redis的连接信息，所以需要重新初始化redis连接信息
}

//切换配置文件
func changeConfCMD() {
	db.InitRedisInfo(true)
}

//添加配置文件
func addConfCMD() {
	conf.CreateRedisConfFile()
}

//切换操作数据
func changeOptDbIdCMD(cmdParams []string) {
	if !checkCMDParamsCount(cmdParams, 2) {
		log.Println("切换操作数据库需要两个参数，请重新输入")
		return
	}
	dbid, err := strconv.Atoi(cmdParams[1])
	if err != nil || dbid <= 0 {
		log.Println("无法解析您输入的数据库编号")
		return
	}
	db.ChangeRedisOptionDBId(dbid)
}
