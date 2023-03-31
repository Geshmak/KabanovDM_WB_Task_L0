package main

import (
	"context"
	"dkab_wb0_m/model"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"

	"github.com/jackc/pgx/v4"
	"github.com/nats-io/stan.go"
)

func reshelp(str string, err error) {
	if err != nil {
		fmt.Printf("Провал - %s (%v)\n", str, err)
	} else {
		fmt.Printf("Успех - %s\n", str)
	}
}

func main() {

	cache := make(map[string]model.Data)
	name := "postgres"
	password := "pas777"

	connection, err := pgx.Connect(context.Background(), fmt.Sprintf("postgres://%s:%s@localhost:5432/postgres?sslmode=disable", name, password))
	reshelp("Подключение к БД", err)
	defer connection.Close(context.Background())

	rows, err := connection.Query(context.Background(), "select data from dkab_test")
	reshelp("Чтение из БД", err)

	for rows.Next() {
		var bytes []byte
		var data model.Data

		err = rows.Scan(&bytes)
		if err != nil {
			fmt.Println(err)
		}
		err = json.Unmarshal(bytes, &data)
		if err != nil {
			fmt.Println(err)
		}

		cache[data.OrderUid] = data
	}
	defer rows.Close()

	sc, err := stan.Connect("test-cluster", "Dkab", stan.NatsURL("nats://localhost:4222"))
	if err != nil && err != io.EOF {
		fmt.Printf("Провал - Подключение к NATS-streaming (%v)\n", err)
	} else {
		fmt.Printf("Успех - Подключение к NATS-streaming\n")
	}
	defer sc.Close()

	_, err = sc.Subscribe("Subj", func(m *stan.Msg) {
		var myData model.Data
		err := json.Unmarshal(m.Data, &myData)
		reshelp("Получение данных из NATS-streaming", err)

		_, ok := cache[myData.OrderUid]
		if !ok && err == nil {

			cache[myData.OrderUid] = myData
			_, err = connection.Exec(context.Background(), "insert into dkab_test values ($1, $2)", myData.OrderUid, m.Data)
			reshelp("Добавление order_uid("+myData.OrderUid+") в БД\n\n", err)
		} else if err == nil {
			fmt.Printf("Данные уже существуют в БД\n\n")
		}
	}, stan.StartWithLastReceived())
	reshelp("Подписка на NATS-streaming", err)

	http.HandleFunc("/", func(respw http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case "GET":
			ParsedF, err := template.ParseFiles("interface.html")
			if err != nil {
				http.Error(respw, err.Error(), 400)
				return
			}

			err = ParsedF.Execute(respw, nil)
			if err != nil {
				http.Error(respw, err.Error(), 400)
				return
			}

		case "POST":
			if val, ok := cache[req.PostFormValue("order_uid")]; ok {

				b, err := json.MarshalIndent(val, "", "\t")
				if err != nil {
					fmt.Println(err)
				}
				fmt.Printf("Данные отправленны, order_uid: %s\n", req.PostFormValue("order_uid"))
				fmt.Fprint(respw, string(b))
			} else {
				fmt.Println("данные с таким order_uid отсутствуют")
				fmt.Fprint(respw, "данные с таким order_uid отсутствуют")
			}
		}
	})
	log.Fatal(http.ListenAndServe("localhost:2023", nil))

}
