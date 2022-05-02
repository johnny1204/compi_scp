package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	ini "gopkg.in/ini.v1"

	"github.com/PuerkitoBio/goquery"
	"github.com/sclevine/agouti"
)

type Config struct {
	Email string
	Pass  string
}

var config Config

func init() {
	cfg, err := ini.Load("config.ini")
	if err != nil {
		fmt.Printf("Fail to read file: %v", err)
		os.Exit(1)
	}

	config = Config{
		Email: cfg.Section("gokuuma").Key("email").String(),
		Pass:  cfg.Section("gokuuma").Key("password").String(),
	}
}

func main() {
	// driver := agouti.ChromeDriver(agouti.Browser("chrome"))
	driver := agouti.ChromeDriver(
		agouti.ChromeOptions("args", []string{
			"--headless",             // headlessモードの指定
			"--window-size=1280,800", // ウィンドウサイズの指定
		}),
		agouti.Debug,
	)
	if err := driver.Start(); err != nil {
		log.Fatalf("Failed to start driver:%v", err)
	}
	defer driver.Stop()

	page, err := driver.NewPage()
	if err != nil {
		log.Fatalf("Failed to open page:%v", err)
	}

	execLogin(page, config.Email, config.Pass)
}

func execLogin(page *agouti.Page, id string, password string) *agouti.Page {
	if err := page.Navigate("https://p.nikkansports.com/goku-uma/membership/login/index.zpl"); err != nil {
		log.Fatalf("Failed to navigate:%v", err)
	}

	fmt.Println("visit login page")
	time.Sleep(2 * time.Second)

	if err := page.FindByXPath("/html/body/div[1]/div[2]/div[1]/section/section/section/div[1]/div/iframe").SwitchToFrame(); err != nil {
		log.Fatalf("Failed to iframe", err)
	}

	page.FindByXPath("/html/body/app-main/app-widget/screen-layout/main/current-screen/div/screen-login/p[3]/input").Fill(id)
	page.FindByXPath("/html/body/app-main/app-widget/screen-layout/main/current-screen/div/screen-login/p[4]/input").Fill(password)

	if err := page.FindByXPath("/html/body/app-main/app-widget/screen-layout/main/current-screen/div/screen-login/p[6]/button").Click(); err != nil {
		log.Fatalf("Failed to login:%v", err)
	}

	time.Sleep(3 * time.Second)

	fmt.Println("login success")

	return page
}

func getCurrentDom(page *agouti.Page) *goquery.Document {
	html, _ := page.HTML()
	readerCurContents := strings.NewReader(html)
	contentsDom, _ := goquery.NewDocumentFromReader(readerCurContents)

	return contentsDom
}
