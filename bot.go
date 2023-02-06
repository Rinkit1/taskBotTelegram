package main

import (
	"bytes"
	"context"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"

	tgbotapi "github.com/skinass/telegram-bot-api/v5"
)

var (
	BotToken   = "5575126796:AAEd7MMklvS9zVKne6Gi-XVdgFVsL6fUbFY"
	WebhookURL = "https://0a0d-94-29-27-238.eu.ngrok.io"
)

type Data struct {
	IDTask     int
	IDOwner    int64
	TextTask   string
	IDWorker   int64
	NameOwner  string
	NameWorker string
}

type DataSet struct {
	Data []Data
}

type InfoTemplate struct {
	Prefix    string
	Number    string
	TextTask  string
	NameOwner string
	WhoWorker string
	Actions   []Action
	EndText   string
}
type Action struct {
	NameAction string
	Number     string
}

const (
	TaskWithoutWorker int64 = -1
	TextTemplate1           = `{{.Prefix}}{{.Number}}. {{.TextTask}} by @{{.NameOwner}}{{.WhoWorker}}{{range $ind, $key := .Actions}}{{$key.NameAction}}{{$key.Number}}{{end}}`
	TextTemplate2           = `Задача "{{.TextTask}}" {{.EndText}}{{.Number}}{{.WhoWorker}}`
)

func startTaskBot(_ context.Context) error {
	bot, err := tgbotapi.NewBotAPI(BotToken)
	if err != nil {
		log.Fatalf("NewBotAPI failed: %s", err)
	}
	bot.Debug = true
	wh, err := tgbotapi.NewWebhook(WebhookURL)
	if err != nil {
		log.Fatalf("NewWebhook failed: %s", err)
	}
	_, err = bot.Request(wh)
	if err != nil {
		log.Fatalf("SetWebhook failed: %s", err)
	}
	updates := bot.ListenForWebhook("/")
	go func() {
		log.Fatalln("http err:", http.ListenAndServe(":8081", nil))
	}()

	dataset := new(DataSet)
	numberOfLastTask := 0

	for update := range updates {
		tagMessage := update.Message.Command()
		switch {
		case tagMessage == "tasks":
			err = taskLogic(bot, update, dataset)
		case tagMessage == "new":
			numberOfLastTask++
			err = newLogic(bot, numberOfLastTask, update, dataset)
		case strings.Contains(update.Message.Text, "/assign_"):
			err = assignLogic(bot, update, dataset)
		case strings.Contains(update.Message.Text, "/unassign_"):
			err = unassignLogic(bot, update, dataset)
		case strings.Contains(update.Message.Text, "/resolve_"):
			err = resolveLogic(bot, update, dataset)
		case tagMessage == "my":
			err = myLogic(bot, update, dataset)
		case tagMessage == "owner":
			err = ownerLogic(bot, update, dataset)
		}
	}
	return err
}

func main() {
	err := startTaskBot(context.Background())
	if err != nil {
		panic(err)
	}
}

func taskLogic(bot *tgbotapi.BotAPI, update tgbotapi.Update, dataset *DataSet) (err error) {
	var text string
	if len(dataset.Data) == 0 {
		text = "Нет задач"
	} else {
		infoTemp := make([]InfoTemplate, 0)
		for ind, info := range dataset.Data {
			var endText, pref string
			actions := make([]Action, 0)
			taskID := strconv.Itoa(info.IDTask)
			switch {
			case update.Message.Chat.ID == info.IDWorker:
				endText = "\nassignee: я"
				act1 := Action{
					NameAction: "\n/unassign_",
					Number:     taskID,
				}
				act2 := Action{
					NameAction: " /resolve_",
					Number:     taskID,
				}
				actions = append(actions, act1, act2)
			case info.IDWorker != TaskWithoutWorker && update.Message.Chat.ID != info.IDWorker:
				endText = "\nassignee: @" + info.NameWorker
			case info.IDWorker == TaskWithoutWorker:
				act1 := Action{
					NameAction: "\n/assign_",
					Number:     taskID,
				}
				actions = append(actions, act1)
			}
			if ind != 0 {
				pref = "\n\n"
			}
			temp := InfoTemplate{
				Prefix:    pref,
				Number:    taskID,
				TextTask:  info.TextTask,
				NameOwner: info.NameOwner,
				WhoWorker: endText,
				Actions:   actions,
			}
			infoTemp = append(infoTemp, temp)
		}
		text = templateToString(infoTemp, TextTemplate1)
	}
	msg := tgbotapi.NewMessage(
		update.Message.Chat.ID,
		text,
	)
	_, err = bot.Send(msg)
	if err != nil {
		log.Fatalf("TaskLogic: Bot Send Message: %s", err)
	}
	return err
}
func newLogic(bot *tgbotapi.BotAPI, number int, update tgbotapi.Update, dataset *DataSet) (err error) {
	var text string
	data := Data{
		IDTask:    number,
		IDOwner:   update.Message.Chat.ID,
		TextTask:  update.Message.CommandArguments(),
		IDWorker:  TaskWithoutWorker,
		NameOwner: update.Message.Chat.UserName,
	}
	dataset.Data = append(dataset.Data, data)
	infoTemp := make([]InfoTemplate, 0)
	endText := "создана, id="
	temp := InfoTemplate{
		TextTask: data.TextTask,
		EndText:  endText,
		Number:   strconv.Itoa(data.IDTask),
	}
	infoTemp = append(infoTemp, temp)
	text = templateToString(infoTemp, TextTemplate2)
	msg := tgbotapi.NewMessage(
		update.Message.Chat.ID,
		text,
	)
	_, err = bot.Send(msg)
	if err != nil {
		log.Fatalf("NewLogic: Bot Send Message: %s", err)
	}
	return err
}
func assignLogic(bot *tgbotapi.BotAPI, update tgbotapi.Update, dataset *DataSet) (err error) {
	taskInfo := strings.Split(update.Message.Command(), "_")
	for ind, info := range dataset.Data {
		if strconv.Itoa(info.IDTask) == taskInfo[1] {
			infoTemp := make([]InfoTemplate, 0)
			endText := "назначена на вас"
			temp := InfoTemplate{
				TextTask: info.TextTask,
				EndText:  endText,
			}
			infoTemp = append(infoTemp, temp)
			text := templateToString(infoTemp, TextTemplate2)
			msg1 := tgbotapi.NewMessage(
				update.Message.Chat.ID,
				text,
			)
			_, err = bot.Send(msg1)
			if err != nil {
				log.Fatalf("AssignLogic: Bot Send Message: %s", err)
				break
			}
			if info.IDWorker != TaskWithoutWorker {
				infoTemp1 := make([]InfoTemplate, 0)
				endText = "назначена на @"
				temp = InfoTemplate{
					TextTask:  info.TextTask,
					EndText:   endText,
					WhoWorker: update.Message.Chat.UserName,
				}
				infoTemp1 = append(infoTemp1, temp)
				text = templateToString(infoTemp1, TextTemplate2)
				msg2 := tgbotapi.NewMessage(
					info.IDWorker,
					text,
				)
				_, err = bot.Send(msg2)
				if err != nil {
					log.Fatalf("AssignLogic: Bot Send Message: %s", err)
					break
				}
			} else if info.IDWorker == TaskWithoutWorker && update.Message.Chat.ID != info.IDOwner {
				infoTemp1 := make([]InfoTemplate, 0)
				endText = "назначена на @"
				temp = InfoTemplate{
					TextTask:  info.TextTask,
					EndText:   endText,
					WhoWorker: update.Message.Chat.UserName,
				}
				infoTemp1 = append(infoTemp1, temp)
				text = templateToString(infoTemp1, TextTemplate2)
				msg2 := tgbotapi.NewMessage(
					info.IDOwner,
					text,
				)
				_, err = bot.Send(msg2)
				if err != nil {
					log.Fatalf("AssignLogic: Bot Send Message: %s", err)
					break
				}
			}
			dataset.Data[ind].IDWorker = update.Message.Chat.ID
			dataset.Data[ind].NameWorker = update.Message.Chat.UserName
		}
	}
	return err
}
func unassignLogic(bot *tgbotapi.BotAPI, update tgbotapi.Update, dataset *DataSet) (err error) {
	var text string
	taskInfo := strings.Split(update.Message.Command(), "_")
	for ind, info := range dataset.Data {
		if strconv.Itoa(info.IDTask) == taskInfo[1] {
			if info.IDWorker != update.Message.Chat.ID {
				msg1 := tgbotapi.NewMessage(
					update.Message.Chat.ID,
					"Задача не на вас",
				)
				_, err = bot.Send(msg1)
				if err != nil {
					log.Fatalf("UnassignLogic: Bot Send Message: %s", err)
					break
				}
			} else {
				msg1 := tgbotapi.NewMessage(
					update.Message.Chat.ID,
					"Принято",
				)
				_, err = bot.Send(msg1)
				if err != nil {
					log.Fatalf("UnassignLogic: Bot Send Message: %s", err)
					break
				}
				infoTemp := make([]InfoTemplate, 0)
				endText := "осталась без исполнителя"
				temp := InfoTemplate{
					TextTask: info.TextTask,
					EndText:  endText,
				}
				infoTemp = append(infoTemp, temp)
				text = templateToString(infoTemp, TextTemplate2)
				msg2 := tgbotapi.NewMessage(
					info.IDOwner,
					text,
				)
				_, err = bot.Send(msg2)
				if err != nil {
					log.Fatalf("UnassignLogic: Bot Send Message: %s", err)
					break
				}
				dataset.Data[ind].IDWorker = TaskWithoutWorker
			}
		}
	}
	return err
}
func resolveLogic(bot *tgbotapi.BotAPI, update tgbotapi.Update, dataset *DataSet) (err error) {
	var text string
	taskInfo := strings.Split(update.Message.Command(), "_")
	for ind, info := range dataset.Data {
		if strconv.Itoa(info.IDTask) == taskInfo[1] {
			if info.IDWorker == update.Message.Chat.ID {
				infoTemp := make([]InfoTemplate, 0)
				endText := "выполнена"
				temp := InfoTemplate{
					TextTask: info.TextTask,
					EndText:  endText,
				}
				infoTemp = append(infoTemp, temp)
				text = templateToString(infoTemp, TextTemplate2)
				msg1 := tgbotapi.NewMessage(
					update.Message.Chat.ID,
					text,
				)
				_, err = bot.Send(msg1)
				if err != nil {
					log.Fatalf("ResolveLogic: Bot Send Message: %s", err)
					break
				}
				infoTemp1 := make([]InfoTemplate, 0)
				endText = "выполнена @"
				temp = InfoTemplate{
					TextTask:  info.TextTask,
					EndText:   endText,
					WhoWorker: update.Message.Chat.UserName,
				}
				infoTemp1 = append(infoTemp1, temp)
				text = templateToString(infoTemp1, TextTemplate2)
				msg2 := tgbotapi.NewMessage(
					info.IDOwner,
					text,
				)
				_, err = bot.Send(msg2)
				if err != nil {
					log.Fatalf("ResolveLogic: Bot Send Message: %s", err)
					break
				}
			} else {
				msg1 := tgbotapi.NewMessage(
					update.Message.Chat.ID,
					"Вы не исполнитель задачи",
				)
				_, err = bot.Send(msg1)
				if err != nil {
					log.Fatalf("ResolveLogic: Bot Send Message: %s", err)
					break
				}
			}
			dataset.Data = append(dataset.Data[:ind], dataset.Data[ind+1:]...)
		}
	}
	return err
}
func myLogic(bot *tgbotapi.BotAPI, update tgbotapi.Update, dataset *DataSet) (err error) {
	var text string
	flag := false
	infoTemp := make([]InfoTemplate, 0)
	for _, info := range dataset.Data {
		if info.IDWorker == update.Message.Chat.ID {
			var pref string
			actions := make([]Action, 0)
			taskID := strconv.Itoa(info.IDTask)
			if flag {
				pref = "\n\n"
			} else {
				flag = true
			}
			act1 := Action{
				NameAction: "\n/unassign_",
				Number:     taskID,
			}
			act2 := Action{
				NameAction: " /resolve_",
				Number:     taskID,
			}
			actions = append(actions, act1, act2)
			temp := InfoTemplate{
				Prefix:    pref,
				Number:    taskID,
				TextTask:  info.TextTask,
				NameOwner: info.NameOwner,
				Actions:   actions,
			}
			infoTemp = append(infoTemp, temp)
		}
	}
	text = templateToString(infoTemp, TextTemplate1)
	msg1 := tgbotapi.NewMessage(
		update.Message.Chat.ID,
		text,
	)
	_, err = bot.Send(msg1)
	if err != nil {
		log.Fatalf("MyLogic: Bot Send Message: %s", err)
	}
	return err
}
func ownerLogic(bot *tgbotapi.BotAPI, update tgbotapi.Update, dataset *DataSet) (err error) {
	var text string
	flag := false
	infoTemp := make([]InfoTemplate, 0)
	for _, info := range dataset.Data {
		if info.IDOwner == update.Message.Chat.ID {
			var pref string
			actions := make([]Action, 0)
			taskID := strconv.Itoa(info.IDTask)
			if info.IDWorker != update.Message.Chat.ID {
				act1 := Action{
					NameAction: "\n/assign_",
					Number:     taskID,
				}
				actions = append(actions, act1)
			} else {
				act1 := Action{
					NameAction: "\n/unassign_",
					Number:     taskID,
				}
				act2 := Action{
					NameAction: " /resolve_",
					Number:     taskID,
				}
				actions = append(actions, act1, act2)
			}
			if flag {
				pref = "\n\n"
			} else {
				flag = true
			}
			temp := InfoTemplate{
				Prefix:    pref,
				Number:    taskID,
				TextTask:  info.TextTask,
				NameOwner: info.NameOwner,
				Actions:   actions,
			}
			infoTemp = append(infoTemp, temp)
		}
	}
	text = templateToString(infoTemp, TextTemplate1)
	msg1 := tgbotapi.NewMessage(
		update.Message.Chat.ID,
		text,
	)
	_, err = bot.Send(msg1)
	if err != nil {
		log.Fatalf("OwnerLogic: Bot Send Message: %s", err)
	}
	return err
}
func templateToString(infoTemp []InfoTemplate, textTemplate string) (text string) {
	t, err := template.New("template").Parse(textTemplate)
	if err != nil {
		log.Fatalf("Template to string: %s", err)
		return ""
	}
	var tpl bytes.Buffer
	for _, val := range infoTemp {
		err = t.Execute(&tpl, val)
		if err != nil {
			log.Fatalf("Template to string: %s", err)
			return ""
		}
	}
	text = tpl.String()
	return text
}
