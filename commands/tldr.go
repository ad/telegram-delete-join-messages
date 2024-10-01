package commands

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"slices"
	"unicode/utf8"

	"github.com/PuerkitoBio/goquery"
	"github.com/tidwall/gjson"
	"mvdan.cc/xurls/v2"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Send Yandex 300 response on link
func (c *Commands) TLDR(ctx context.Context, b *bot.Bot, update *models.Update) {
	if c.config.YandexToken == "" {
		return
	}

	if !slices.Contains(c.config.AllowedChatIDsList, update.Message.Chat.ID) {
		return
	}

	rxStrict := xurls.Strict()

	link := rxStrict.FindString(update.Message.Text)
	if link == "" {
		_, errSendMessage := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    update.Message.Chat.ID,
			Text:      "Бот заберёт статью по ссылке и сделает её краткое описание.",
			ParseMode: models.ParseModeHTML,
		})

		if errSendMessage != nil {
			fmt.Println("errSendMessage (/tldr): ", errSendMessage)
		}

		return
	}

	client := &http.Client{}
	req, err := http.NewRequest("POST", "https://300.ya.ru/api/sharing-url",
		bytes.NewBuffer([]byte(`{"article_url": "`+link+`"}`)))
	if err != nil {
		return
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "OAuth "+c.config.YandexToken)

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	if gjson.Get(string(body), "status").Str != "success" {
		return
	}

	res, err := http.Get(gjson.Get(string(body), "sharing_url").Str)
	if err != nil {
		return
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return
	}

	text := doc.Find(".summary .summary-content .summary-text").Text()

	text = regexp.MustCompile(`\n\s+\n|\n\n`).ReplaceAllString(text, "\n")
	text = regexp.MustCompile(`[ ]+`).ReplaceAllString(text, ` `)

	if utf8.RuneCountInString(text) > 4000 {
		text = string([]rune(text)[:4000])
	}

	_, errSendMessage := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      text,
		ParseMode: models.ParseModeHTML,
	})

	if errSendMessage != nil {
		fmt.Println("errSendMessage (/tldr): ", errSendMessage)
	}
}
