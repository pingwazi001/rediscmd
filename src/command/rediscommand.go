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
	"time"

	"github.com/modood/table"
)

var InputReader *bufio.Reader

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
		{Key: "ldb", Value: fmt.Sprintf("加载数据库列表 [y|n] [0~%d)", db.RedisDBCount())},
		{Key: "keys", Value: "模糊查询缓存key [keypattern]"},
		{Key: "get", Value: "查询模糊key的值 [keypattern]"},
		{Key: "del", Value: "删除模糊key的值 [keypattern]"},
		{Key: "set", Value: "设置精确key的值 [key] [value]"},
		{Key: "resetconf", Value: fmt.Sprintf("重新配置%s文件内容", conf.RedisConfName())},
		{Key: "changeconf", Value: "切换配置文件"},
		{Key: "reloadkeys", Value: "触发刷新本地缓存Key集合"},
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
	case "ldb":
		loadDBCMD(cmdParams)
	case "keys":
		keysCMD(cmdParams)
	case "get":
		getCMD(cmdParams)
	case "del":
		delCMD(cmdParams)
	case "set":
		setCMD(cmdParams)
	case "resetconf":
		resetConfCMD()
	case "changeconf":
		changeConfCMD()
	case "reloadkeys":
		refreshLocalKeysCMD()
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
	var dbInfoChan chan model.RedisDBInfo
	loadDbCount := db.RedisDBCount()
	switch opt {
	case "y":
		log.Println("正在加载全部数据库信息，请稍候...")
		dbInfoChan = db.AllRedisDBInfo(true, 0)
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
		log.Println("正在加载数据库信息，请稍候...")
		dbInfoChan = db.AllRedisDBInfo(false, count)
	default:
		log.Println("加载方式仅允许输入y/n")
		return
	}
	dbMaps := make(map[int]int, loadDbCount)
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

//加载缓存key
func keysCMD(cmdParams []string) {
	if !checkCMDParamsCount(cmdParams, 2) {
		log.Println("模糊查询缓存Key的列表需要两个参数，请重新输入")
		return
	}
	keysChan, err := db.SearchRedisKeys(cmdParams[1])
	if err != nil {
		log.Println(err)
		return
	}
	for i := 0; i < len(keysChan); i++ {
		keys := <-keysChan
		for _, item := range keys {
			item = strings.ReplaceAll(item, " ", "")
			if item == "" {
				continue
			}
			log.Println(item)
		}
	}
}

//获取指定key的值
func getCMD(cmdParams []string) {
	if !checkCMDParamsCount(cmdParams, 2) {
		log.Println("模糊查询缓存Key的值需要两个参数，请重新输入")
		return
	}
	keysChan, err := db.SearchRedisKeys(cmdParams[1])
	if err != nil {
		log.Println(err)
		return
	}
	keysCount := 0
	valueMsgChan := make(chan string, len(keysChan))
	for i := 0; i < len(keysChan); i++ {
		keys := <-keysChan
		if len(keys) <= 0 {
			continue
		}
		keysCount += len(keys)
		for _, key := range keys {
			go func(itemKey string) {
				value, err := db.GetRedisValue(itemKey)
				if err != nil {
					valueMsgChan <- itemKey + "=" + err.Error()
					return
				}
				valueMsgChan <- fmt.Sprintf("%s=%s", itemKey, value)
			}(key)
		}
	}
	//打印查询结果
	for i := 0; i < keysCount; i++ {
		log.Println(<-valueMsgChan)
		log.Println()
	}
	close(valueMsgChan) //释放资源
}

//删除模糊key的值
func delCMD(cmdParams []string) {
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
	keysChan, err := db.SearchRedisKeys(pattern)
	if err != nil {
		log.Println(err)
		return
	}

	keysBuf := make([]string, 0, 2000)
	keysCount := 0
	delCount := 0
	delKeysCount := 0
	deleteMsgChan := make(chan string, len(keysChan))
	for i := 0; i < len(keysChan); i++ {
		keys := <-keysChan
		if len(keys) <= 0 {
			continue
		}
		delKeysCount += len(keys)
		keysCount += len(keys)
		keysBuf = append(keysBuf, keys...)
		if len(keysBuf) >= 1000 {
			go func(itemKey []string) {
				start := time.Now()
				db.DeleteRedisKey(itemKey...) //一次性批量删除多个
				deleteMsgChan <- fmt.Sprintf("%s 删除成功，耗时%d毫秒", itemKey, time.Since(start).Milliseconds())
			}(keysBuf[0:keysCount])
			keysBuf = keysBuf[0:0]
			keysCount = 0
			delCount++
		}
	}
	for i := 0; i < delCount; i++ {
		log.Println(<-deleteMsgChan)
	}
	log.Printf("共删除%d个缓存", delKeysCount)
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

//触发加载缓存key
func refreshLocalKeysCMD() {
	db.TriggerLoadAllCacheKeysToLocal()
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
