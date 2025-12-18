package main

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		printHelp()
		return
	}

	switch os.Args[1] {
	case "now":
		fmt.Println(time.Now().Unix())
	case "format":
		if len(os.Args) < 3 {
			fmt.Println("Usage: timeutil format <unix_timestamp>")
			return
		}
		ts, err := strconv.ParseInt(os.Args[2], 10, 64)
		if err != nil {
			fmt.Printf("Invalid timestamp: %v\n", err)
			return
		}
		fmt.Println(time.Unix(ts, 0).Format(time.RFC3339))
	case "parse":
		if len(os.Args) < 3 {
			fmt.Println("Usage: timeutil parse <date_string>")
			return
		}
		t, err := time.Parse("2006-01-02 15:04:05", os.Args[2])
		if err != nil {
			fmt.Printf("Invalid date: %v\n", err)
			return
		}
		fmt.Println(t.Unix())
	default:
		printHelp()
	}
}

func printHelp() {
	fmt.Println(`Time Utility - конвертация Unix времени
Commands:
  now                    - текущее время в Unix формате
  format <timestamp>     - преобразовать Unix время в читаемый формат
  parse <date_string>    - преобразовать дату в Unix время (формат: YYYY-MM-DD HH:MM:SS)
  add <timestamp> <sec>  - добавить секунды к времени
  diff <ts1> <ts2>       - разница между двумя временами в секундах`)
}
