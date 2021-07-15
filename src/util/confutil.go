package util

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/gcfg.v1"
)

var (
	confPath = "conf.ini"
)

type Conf struct {
	Redis struct {
		AddRess  string
		Port     int
		Password string
	}
}

//移除特殊字符
func removeSpecialChar(str string) string {
	str = strings.ReplaceAll(str, "\n", "")
	str = strings.ReplaceAll(str, "\r", "")
	return str
}

//初始化配置
func InitConf() error {
	os.Remove(confPath) //删除配置文件
	fileObj, err := os.OpenFile(confPath, os.O_CREATE|os.O_WRONLY, os.ModeAppend|os.ModePerm)
	if err != nil {
		os.Remove(confPath) //文件打开失败，删除配置文件
		return errors.New("初始化配置文件失败，请确保工具所在目录中不存在conf.ini文件，并且此文件未打开")
	}
	defer fileObj.Close()
	writer := bufio.NewWriter(fileObj)
	writer.WriteString("[redis]\n")
	stdInput := bufio.NewReader(os.Stdin)
	fmt.Println("请输入redis连接地址:")
	address, err := stdInput.ReadString('\n')
	address = removeSpecialChar(address)
	if err != nil || address == "" {
		os.Remove(confPath) //地址读取失败，删除配置文件

		return fmt.Errorf("redis地址读取失败，%s", err.Error())
	}

	fmt.Println("请输入redis连接端口:")
	portStr, err := stdInput.ReadString('\n')
	portStr = removeSpecialChar(portStr)
	if err != nil {
		os.Remove(confPath) //端口读取失败，删除配置文件
		return fmt.Errorf("redis端口读取失败，%s", err.Error())
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port == 0 {
		os.Remove(confPath) //端口解析失败，删除配置文件
		return fmt.Errorf("redis端口解析失败，%s", err.Error())
	}
	fmt.Println("请输入redis访问密码:")
	password, err := stdInput.ReadString('\n')
	password = removeSpecialChar(password)
	if err != nil || password == "" {
		os.Remove(confPath) //密码读取失败，删除配置文件
		return fmt.Errorf("redis密码读取失败，%s", err.Error())
	}
	writer.WriteString(fmt.Sprintf("AddRess=%s\n", address))
	writer.WriteString(fmt.Sprintf("Port=%d\n", port))
	writer.WriteString(fmt.Sprintf("Password=%s", password))
	writer.Flush()
	return nil
}

//检查配置文件是否存在
func CheckConf() error {
	//检查配置文件是否存在
	if _, err := os.Stat(confPath); os.IsNotExist(err) {
		fmt.Println("配置文件读取失败")
		return InitConf()
	}
	config := new(Conf)
	err := gcfg.ReadFileInto(config, confPath)
	if err != nil {
		fmt.Println("配置文件读取失败，", err)
		return InitConf()
	}
	if config.Redis.AddRess == "" || config.Redis.Password == "" || config.Redis.Port == 0 {
		fmt.Println("配置信息不正确")
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
	err := gcfg.ReadFileInto(config, confPath)
	if err != nil {
		return nil, err
	}
	return config, nil
}
