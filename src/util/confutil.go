package util

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/gcfg.v1"
)

var (
	confName = "conf.ini"
)

type Conf struct {
	Redis struct {
		AddRess    string
		Port       int
		Password   string
		MaxConnect int //连接池中允许最大的连接数
	}
}

//删除错误的配置文件
func removeConfFile() {
	os.Remove(confName) //文件打开失败，删除配置文件
}

//移除特殊字符
func removeSpecialChar(str string) string {
	str = strings.ReplaceAll(str, "\n", "")
	str = strings.ReplaceAll(str, "\r", "")
	return str
}

//初始化配置
func InitConf() error {
	os.Remove(confName) //删除配置文件
	fileObj, err := os.Create(confName)
	if err != nil {
		os.Remove(confName) //文件打开失败，删除配置文件
		return errors.New("初始化配置文件失败，请确保工具所在目录中不存在conf.ini文件，并且此文件未打开")
	}
	defer fileObj.Close()
	writer := bufio.NewWriter(fileObj)
	writer.WriteString("[redis]\n")
	stdInput := bufio.NewReader(os.Stdin)
	log.Println("请输入redis连接地址:")
	address, err := stdInput.ReadString('\n')
	address = removeSpecialChar(address)
	if err != nil || address == "" {
		os.Remove(confName) //地址读取失败，删除配置文件
		return fmt.Errorf("redis地址读取失败")
	}

	log.Println("请输入redis连接端口:")
	portStr, err := stdInput.ReadString('\n')
	portStr = removeSpecialChar(portStr)
	if err != nil {
		os.Remove(confName) //端口读取失败，删除配置文件
		return fmt.Errorf("redis端口读取失败，%s", err.Error())
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port == 0 {
		os.Remove(confName) //端口解析失败，删除配置文件
		return fmt.Errorf("redis端口解析失败")
	}

	log.Println("请输入redis访问密码:")
	password, err := stdInput.ReadString('\n')
	password = removeSpecialChar(password)
	if err != nil || password == "" {
		os.Remove(confName) //密码读取失败，删除配置文件
		return fmt.Errorf("redis密码读取失败")
	}

	log.Println("请输入redis连接池中允许的最大连接数（1~100）:")
	maxConnectStr, err := stdInput.ReadString('\n')
	maxConnectStr = removeSpecialChar(maxConnectStr)
	if err != nil {
		os.Remove(confName) //端口读取失败，删除配置文件
		return fmt.Errorf("redis连接池中允许的最大连接数读取失败，%s", err.Error())
	}
	maxConnect, err := strconv.Atoi(maxConnectStr)
	if err != nil || maxConnect < 1 || maxConnect > 100 {
		os.Remove(confName) //端口解析失败，删除配置文件
		return fmt.Errorf("redis连接池中允许的最大连接数解析失败")
	}
	writer.WriteString(fmt.Sprintf("AddRess=%s\n", address))
	writer.WriteString(fmt.Sprintf("Port=%d\n", port))
	writer.WriteString(fmt.Sprintf("Password=%s\n", password))
	writer.WriteString(fmt.Sprintf("MaxConnect=%d", maxConnect))
	writer.Flush()
	return nil
}

//检查配置文件是否存在
func CheckConf() error {
	initConfFileName()
	//检查配置文件是否存在
	if _, err := os.Stat(confName); os.IsNotExist(err) {
		log.Printf("%s配置文件读取失败，请初始化此配置信息", confName)
		return InitConf()
	}
	config := new(Conf)
	err := gcfg.ReadFileInto(config, confName)
	if err != nil {
		log.Printf("%s配置文件读取失败，请初始化此配置信息", confName)
		return InitConf()
	}
	if config.Redis.AddRess == "" || config.Redis.Password == "" || config.Redis.Port == 0 || config.Redis.MaxConnect < 1 || config.Redis.MaxConnect > 100 {
		log.Printf("%s配置内置内容不正确，请初始化此配置信息", confName)
		return InitConf()
	}
	return nil
}

//获取配置
func GetConf() (*Conf, error) {
	checkErr := CheckConf()
	if checkErr != nil {
		return nil, checkErr
	}
	config := new(Conf)
	err := gcfg.ReadFileInto(config, confName)
	if err != nil {
		return nil, err
	}
	return config, nil
}

//初始化配置文件名词
func initConfFileName() {
	pwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	confFileNames, err := filepath.Glob(filepath.Join(pwd, "conf*.ini"))
	if err != nil {
		log.Fatal(err)
	}
	if len(confFileNames) <= 0 {
		return
	}
	fileNames := []string{}
	for _, fileName := range confFileNames {
		_, name := filepath.Split(fileName)
		fileNames = append(fileNames, name)
	}

	//只有一个配置文件，就不用选择了，直接使用
	if len(fileNames) == 1 {
		confName = fileNames[0]
		return
	}
	for index, fileName := range fileNames {
		log.Printf("%d %s\r\n", index, fileName)
	}
	log.Printf("您当前存在多个配置文件，请输入您要使用的文件编号[0~%d):", len(fileNames))
	stdInput := bufio.NewReader(os.Stdin)
	fileNOStr, _ := stdInput.ReadString('\n')
	fileNOStr = removeSpecialChar(fileNOStr)
	var fileNO int
	var atoiErr error
	if fileNO, atoiErr = strconv.Atoi(fileNOStr); atoiErr != nil || fileNO < 0 || fileNO >= len(fileNames) {
		log.Println("您输入的文件编号不在指定范围内，请重新输入")
		initConfFileName()
	}
	confName = fileNames[fileNO]
}

func createConfFile() {
	backConfName := confName
	log.Println("新生成的配置名称名称格式将为【conf-*.ini】，其中*是需要您输入的部分")
	log.Println("如果文件存在，将覆盖之前的文件！！！\r\n如果文件存在，将覆盖之前的文件！！！\r\n如果文件存在，将覆盖之前的文件！！！")
	log.Println("请输入新增配置文件的自定义部分（不能为空或者*）:")
	stdInput := bufio.NewReader(os.Stdin)
	confCustNameStr, _ := stdInput.ReadString('\n')
	confCustNameStr = strings.Trim(removeSpecialChar(confCustNameStr), " ")
	if confCustNameStr == "" || confCustNameStr == "*" {
		log.Println("您输入的配置文件自定义部分格式不正确")
		return
	}
	confName = fmt.Sprintf("conf-%s.ini", confCustNameStr)
	InitConf()              //初始化配置文件
	confName = backConfName //换回目前使用的配置文件
}
