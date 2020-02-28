package main

import (
	"log"
	"os/exec"
)

func main() {
	mosStart := exec.Command("sh", "-c", "docker run -d --rm --name mosquitto eclipse-mosquitto")
	if err := mosStart.Run(); err != nil {
		log.Println(err.Error())
	} else {
		log.Println("docker start success")
	}
}

// type zaka struct{}

// func main() {
// 	zA := zaka{}
// 	zB := &zaka{}
// 	zA.funcA()
// 	zB.funcB()
// }

// func (za *zaka) funcA() {
// 	println("hello")
// }

// func (za zaka) funcB() {
// 	println("world")
// }

// func main() {
// 	chQuit := make(chan struct{})
// 	go func() {
// 		quit := false
// 		for !quit {
// 			select {
// 			case <-chQuit:
// 				quit = true
// 			}
// 		}
// 		println("world")
// 	}()
// 	println("hello")
// 	time.Sleep(time.Second)
// 	close(chQuit)
// 	time.Sleep(time.Second)
// }

// func main() {
// 	defer hello()()
// 	println("world")
// }

// func hello() func() {
// 	println("hello")
// 	return func() {
// 		var l list.List
// 		for i := 0; i < 3; i++ {
// 			x := i
// 			l.PushBack(func() {
// 				println(x)
// 			})
// 		}
// 		for s := l.Back(); s != nil; s = s.Prev() {
// 			if fc, ok := s.Value.(func()); ok {
// 				fc()
// 			}
// 		}
// 	}
// }

// var locker = new(sync.Mutex)
// var cond = sync.NewCond(locker)

// func main() {
// 	for i := 0; i < 10; i++ {
// 		go func(x int) {
// 			cond.L.Lock()
// 			defer cond.L.Unlock()
// 			cond.Wait()

// 			fmt.Println(x)
// 		}(i)
// 	}

// 	time.Sleep(time.Second)
// 	fmt.Println("Signal ...")
// 	cond.Signal()
// 	time.Sleep(time.Second)
// 	fmt.Println("Signal ...")
// 	cond.Signal()
// 	time.Sleep(time.Second)
// 	fmt.Println("Broadcast ...")
// 	cond.Broadcast()
// 	time.Sleep(time.Second)
// }

// type res struct{}

// var _res res

// func (*res) hello() {
// 	println("hello")
// }

// func do(some func()) {
// 	some()
// }

// func main() {
// 	do((&_res).hello)
// }

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
