package main

import (
	"rediscmd/src/command"
)

//dlv debug --headless --listen=:2345 --log --api-version=2
func main() {
	command.RedisCMDStart()
}
