package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	// пайплайн артефактов
	c := make(chan artifact)
	c1 := make(chan artifact)
	c2 := make(chan artifact)
	art := artifact{}

	bot, err := tgbotapi.NewBotAPI("myAwesomeApi")
	if err != nil {
		log.Panic(err)
	}
	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			switch update.Message.Text {
			case "/common":
				common := createCommon(art, c) //не объявлять в начале кода! внутри функции рандома
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, common)
				msg.ParseMode = "markdown"
				bot.Send(msg)
			case "/uncommon":
				uncommon := createUncommon(art, c, c1)
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, uncommon)
				msg.ParseMode = "markdown"
				bot.Send(msg)
			case "/rare":
				rare := createRare(art, c, c1, c2)
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, rare)
				msg.ParseMode = "markdown"
				bot.Send(msg)
			case "/random 3":
				var text string
				for i := 0; i < 3; i++ {
					rand.Seed(time.Now().UnixNano())
					rand := rand.Intn(3)
					switch rand {
					case 0:
						common := createCommon(art, c)
						text += common
					case 1:
						uncommon := createUncommon(art, c, c1)
						text += uncommon
					case 2:
						rare := createRare(art, c, c1, c2)
						text += rare
					}
					text += "\n"
				}
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
				msg.ParseMode = "markdown"
				bot.Send(msg)
			default:
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Неизвестная команда")
				bot.Send(msg)
			}
		}
	}
}

type artifact struct {
	item
	description   string
	preAttribute  string // Атрибут перед названием предмета
	postAttribute string // Атрибут после названия предмета
}
type item struct {
	itemName  string
	itemGenus string
}

// ГЕНЕРАЦИЯ ПРЕДМЕТОВ
// создать обычный предмет
func createCommon(art artifact, c chan artifact) string {
	go art.newItem(c)
	art = <-c
	msg := fmt.Sprintf("*" + art.itemName + "*" + ". " + art.description) //формирование markdown сообщения для вывода
	return msg
}

// создать необычный предмет (не помню почему именно так, возможно просто моё баловство)
var createUncommon = func(art artifact, c chan artifact, c1 chan artifact) string {
	go art.newItem(c)
	go newPostAtt(c, c1)
	art = <-c1
	msg := fmt.Sprintf("*" + art.itemName + " " + art.postAttribute + "*" + ". " + art.description)
	return msg
}

// создать редкий предмет
var createRare = func(art artifact, c chan artifact, c1 chan artifact, c2 chan artifact) string {
	go art.newItem(c)
	go newPostAtt(c, c1)
	go newPreAtt(c1, c2)
	art = <-c2
	msg := fmt.Sprintf("*" + art.preAttribute + " " + strings.ToLower(art.itemName) + " " + art.postAttribute + "*" + ". " + art.description)
	return msg
}

// РАБОТА С ОБЪЕКТАМИ
const (
	itemsList         = "./lists/items.csv"         // список предметов с описаниями и указанием на родовое окончание
	postAttributeList = "./lists/postAttribute.csv" // список  пост-атрибутов (после названия) с описаниями
	preAttributeList  = "./lists/preAttribute.csv"  // список пре-атрибутов (перед названием) с описаниями и указанием на тип склонения
	engingsList       = "./lists/engings.csv"       // список окончаний прилагательных
)

// читаем csv
func dump(list string) [][]string {
	file, err := os.Open(list)
	defer file.Close()
	if err != nil {
		panic(err)
	}
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		log.Fatal(err)
	}
	return records
}

// функция для рандомного выбора объекта из любого списка
func randChoice(records [][]string) int {
	rand.Seed(time.Now().UnixNano())
	randRow := rand.Intn(len(records))
	return randRow
}

// Новый предмет в структуре артефакт
func (art artifact) newItem(c chan artifact) {
	records := dump(itemsList)
	name := randChoice(records)                       // выбор рандомного предмета:
	art.itemName = records[name][0]                   // название (палка, монокль)
	art.itemGenus = records[name][1]                  // род — нужен для прилагательного
	randColumn := rand.Intn(len(records[name])-2) + 2 // Рандомное описание предмета. Первые две колонки всегда зарезервированы — так делать скорее неправильно,
	art.description = records[name][randColumn]       // но мы отталкиваемся от того, что так удобнее геймдизайнерам
	c <- art                                          //предмет готов — передаём на конвейер
}

// Добавление пост-атрибута в артефакт
func newPostAtt(c chan artifact, c1 chan artifact) {
	records := dump(postAttributeList)
	name := randChoice(records)
	art := <-c
	art.postAttribute = records[name][0]
	art.description += fmt.Sprintf(" " + records[name][1])
	c1 <- art
}

// Добавление пре-атрибута в артефакт
func newPreAtt(c1 chan artifact, c2 chan artifact) {
	records := dump(preAttributeList)
	name := randChoice(records)                         //выбираем прилагательное
	declension, _ := strconv.Atoi(records[name][1])     //определяем склонение
	art := <-c1                                         // забираем предмет с конвейера
	ending := endingGenerate(art.itemGenus, declension) //получаем окончание
	art.preAttribute = fmt.Sprintf(records[name][0] + ending)
	art.description += fmt.Sprintf(" " + records[name][2])
	c2 <- art
}

// генерация родового окончания
func endingGenerate(genus string, declension int) string {
	records := dump(engingsList)
	var ending string
	switch genus {
	case "ж":
		ending = records[declension][1]
	case "с":
		ending = records[declension][2]
	case "мн":
		ending = records[declension][3]
	default:
		ending = records[declension][0]
	}
	return ending
}
