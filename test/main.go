package main

type res struct{}

var _res res

func (*res) hello() {
	println("hello")
}

func do(some func()) {
	some()
}

func main() {
	do((&_res).hello)
}

// func main() {
// 	println("HELLO WORLD!")
// }

// func main() {
// 	println(base64.StdEncoding.EncodeToString([]byte("tcp://localhost:1883")))
// 	println(base64.StdEncoding.EncodeToString([]byte("#")))
// }

// func main() {
// 	go func() {
// 		defer func() {
// 			if err := recover(); err != nil {
// 				// log.Println(err)
// 				log.Printf("%#v", err)
// 			}
// 			println("defer ...")
// 		}()
// 		println("doing ...")
// 		panic("error ...")
// 	}()
// 	time.Sleep(2 * time.Second)
// 	println("done ...")
// }

// func main() {
// 	println("START")
// 	quit := make(chan os.Signal)
// 	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
// 	toQuit := false
// 	for {
// 		if toQuit {
// 			break
// 		}
// 		tick := time.Tick(time.Second)
// 		select {
// 		case <-quit:
// 			toQuit = true
// 			println("END")
// 			break
// 		case <-tick:
// 			println("---")
// 		}
// 	}
// }

// func main() {
// 	x := 1
// 	for {
// 		time.Sleep(time.Second)
// 		println(x)
// 		x++
// 		if x == 5 {
// 			break
// 		}
// 	}
// }

// func main() {
// 	println("START")
// 	quit := make(chan os.Signal)
// 	signal.Notify(quit, os.Interrupt, os.Kill)

// 	go func() {
// 		for {
// 			time.Sleep(1 * time.Second)
// 			println("---")
// 		}
// 	}()
// 	select {
// 	case <-quit:
// 		time.Sleep(2 * time.Second)
// 		println("END")
// 	}
// }
