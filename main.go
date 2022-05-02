package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
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

	if err := page.Navigate("https://p.nikkansports.com/goku-uma/member/compi/compi_list.zpl"); err != nil {
		log.Fatalf("Failed to compi page:%v", err)
	}

	dom := getCurrentDom(page)
	dom.Find("#compiArea > ol > li:nth-child(1) > a").Each(func(idx int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		compi(page, "https://p.nikkansports.com/goku-uma/member/compi/"+href)
	})
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

// コンピページ
func compi(page *agouti.Page, targetUrl string) {
	if err := page.Navigate(targetUrl); err != nil {
		log.Fatalf("Failed to compi page:%v", err)
	}

	fmt.Println("visit " + targetUrl)
	time.Sleep(3 * time.Second)

	contentsDom := getCurrentDom(page)
	// listDom := contentsDom.Find("#bySchedule > ul.dateList")
	// listLen := listDom.Length()

	var records = []map[int]string{}
	// for i := 1; i <= listLen; i++ {
	// 最新取得
	for i := 1; i <= 1; i++ {
		// for i := 2; i <= 2; i++ {
		is := strconv.Itoa(i)
		contentsDom.Find("#bySchedule > ul.dateList:nth-child(" + is + ") > li > a").Each(func(idx int, s *goquery.Selection) {
			href, _ := s.Attr("href")
			fmt.Println(href)

			records = compiDetail(page, "https://p.nikkansports.com/goku-uma/member/compi/"+href, records)
			time.Sleep(5 * time.Second)
		})
	}

	writeCsv(records)
}

func compiDetail(page *agouti.Page, targetUrl string, records []map[int]string) []map[int]string {
	u, _ := url.Parse(targetUrl)
	query := u.Query()

	if err := page.Navigate(targetUrl); err != nil {
		log.Fatalf("Failed to compi detail page:%v", err)
	}

	html, _ := page.HTML()
	readerCurContents := strings.NewReader(html)
	compiDom, _ := goquery.NewDocumentFromReader(readerCurContents)
	title := compiDom.Find("#contentTit").Text()
	raceDetail := strings.Split(title, "－")[1]
	rn, _ := strconv.Atoi(raceDetail[0:1])

	date := strings.Split(raceDetail, "回")[1][6:]
	dateI, _ := strconv.Atoi(strings.Split(date, "日")[0])

	compiDom.Find("#compiArea > table.compiTable.umabanTable > tbody > tr:nth-child(n + 2)").Each(func(idx int, coms *goquery.Selection) {
		horse := coms.Find("td:nth-child(n + 3)")
		var compiNum = make(map[int]string, horse.Length()+1)
		// CSV用ID
		raceNum := fmt.Sprintf("%02d", idx+1)
		compiNum[0] = query.Get("date") + query.Get("course_id")[1:] + fmt.Sprintf("%02d", rn) + fmt.Sprintf("%02d", dateI) + raceNum
		horse.Each(func(_ int, cs *goquery.Selection) {
			var horseNum string
			cs.Contents().Each(func(i int, s *goquery.Selection) {
				if !s.Is("br") {
					text := strings.TrimSpace(s.Text())
					if text == "消" {
						text = "0"
					}

					if i == 0 {
						horseNum = text
					} else {
						n, _ := strconv.Atoi(horseNum)
						compiNum[n] = text
					}
				}
			})
		})
		records = append(records, compiNum)
	})

	return records
}

func writeCsv(records []map[int]string) {
	file, _ := os.OpenFile("compi.csv", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer file.Close()
	// w := csv.NewWriter(os.Stdout)
	w := csv.NewWriter(file)
	for _, value := range records {
		r := make([]string, 0, 1+len(value))
		for i := 0; i < len(value); i++ {
			r = append(r, value[i])
		}
		if err := w.Write(r); err != nil {
			fmt.Println("error writing record to csv:", err)
		}
	}
	defer w.Flush()
}

func getCurrentDom(page *agouti.Page) *goquery.Document {
	html, _ := page.HTML()
	readerCurContents := strings.NewReader(html)
	contentsDom, _ := goquery.NewDocumentFromReader(readerCurContents)

	return contentsDom
}
