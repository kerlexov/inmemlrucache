package main

import (
	"cache/cache"
	"time"
)

func main() {

	kk := cache.NewCache(5, time.Second)

	kk.Set("1", 1)
	kk.Display()
	kk.Set("2", 2)
	kk.Display()
	kk.Set("3", 3)
	kk.Display()
	kk.Set("4", 4)
	kk.Display()
	kk.Set("3", 3)
	kk.Display()
	kk.Set("5", 5)
	kk.Display()
	kk.Set("8", 8)
	kk.Display()
}
