package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"

	"github.com/nats-io/stan.go"
)

func main() {
	sc, err := stan.Connect("test-cluster", "dkabSend", stan.NatsURL("nats://localhost:4222"))
	if err != nil && err != io.EOF {
		log.Fatalln(err)
	} else {
		log.Println("Успех - Подключение к NATS-streaming")
	}

	var filename string
	for {
		fmt.Println("Введите имя файла")
		fmt.Scanf("%s\n", &filename)
		dataFromFile, _ := ioutil.ReadFile(filename)
		//fmt.Println(dataFromFile)
		sc.Publish("Subj", dataFromFile)
		println("Данные отправлены")
	}
}
