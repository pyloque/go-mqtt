package main

import "github.com/pyloque/mqtt/mqtt"

func main() {
	mqtt.Start("tcp", ":2222")
}
