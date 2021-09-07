package main

import (
	_ "net/http/pprof" //性能分析
	"rediscmd/src/command"
)

//dlv debug --headless --listen=:2345 --log --api-version=2  //控制台调试
//go tool pprof -http=:8081  http://localhost:8080/debug/pprof/profile?seconds=3 //cpu分析
//go tool pprof -http=:8081  http://localhost:8080/debug/pprof/heap?seconds=3    //内存分析
func main() {
	// go func() {
	// 	http.ListenAndServe("0.0.0.0:8080", nil)
	// }()
	command.RedisCMDStart()
}
