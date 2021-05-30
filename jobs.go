package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type job struct {
	id       string
	title    string
	company  string
	location string
	summary  string
	salary   string
}

const baseURL = "https://kr.indeed.com/jobs"

func main() {
	args := os.Args
	if len(args) < 2 {
		fmt.Println("please input industry, job and company name")
		os.Exit(0)
	}

	start := time.Now()
	searchWord := args[1]
	scrapeJob(searchWord)

	fmt.Println("Done", time.Since(start))
}

func scrapeJob(searchWord string) {
	fmt.Printf("search [%s]...\n", searchWord)

	totalJobCounts := getJobCounts(searchWord)
	fmt.Println("total jobs:", totalJobCounts)

	fmt.Println("finding jobs...")
	var jobs []*job
	wg := sync.WaitGroup{}
	for start := 0; start < totalJobCounts; start += 50 {
		wg.Add(1)

		go func(start int) {
			defer wg.Done()
			jobs = append(jobs, getJobs(searchWord, start)...)
		}(start)
	}
	wg.Wait()

	fmt.Println("writing to csv...")
	writeToCSV(jobs, searchWord)
}

func getJobCounts(searchWord string) int {
	url := fmt.Sprintf("%s?q=%s", baseURL, searchWord)

	res, err := http.Get(url)
	checkErr(err)
	defer res.Body.Close()
	checkResp(res)

	doc, err := goquery.NewDocumentFromReader(res.Body)
	checkErr(err)

	pages := 0
	doc.Find("#searchCountPages").Each(func(i int, s *goquery.Selection) {
		// ex) "1페이지 결과 2,195건"
		resultText := cleanString(s.Text())

		r, _ := regexp.Compile("[0-9,]+")
		str := r.FindAllString(resultText, -1)
		if len(str) < 1 {
			return
		}

		cnt := strings.Replace(str[1], ",", "", -1)
		pages, _ = strconv.Atoi(cnt)
	})

	return pages
}

func getJobs(searchWord string, start int) []*job {
	url := fmt.Sprintf("%s?q=%s&limit=50&start=%d", baseURL, searchWord, start)
	fmt.Println("Get job :", url)

	res, err := http.Get(url)
	checkErr(err)
	defer res.Body.Close()
	checkResp(res)

	doc, err := goquery.NewDocumentFromReader(res.Body)
	checkErr(err)

	var jobs []*job
	doc.Find(".jobsearch-SerpJobCard").Each(func(i int, s *goquery.Selection) {
		id, _ := s.Attr("data-jk")
		title := cleanString(s.Find(".title>a").Text())
		company := cleanString(s.Find(".sjcl .company").Text())
		location := cleanString(s.Find(".sjcl .location").Text())
		summary := cleanString(s.Find(".summary").Text())
		salary := cleanString(s.Find(".salary .salaryText").Text())

		jobs = append(jobs, &job{
			id,
			title,
			company,
			location,
			summary,
			salary,
		})
	})

	return jobs
}

func cleanString(str string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(str)), " ")
}

func writeToCSV(jobs []*job, searchWord string) {
	file, err := os.Create("./jobs.csv")
	checkErr(err)

	wr := csv.NewWriter(bufio.NewWriter(file))
	defer wr.Flush()

	headers := []string{"Link", "Title", "Company", "Location", "Summary", "Salary"}
	err = wr.Write(headers)
	checkErr(err)

	for _, job := range jobs {
		link := fmt.Sprintf("%s?q=%s&vjk=%s", baseURL, searchWord, job.id)
		err := wr.Write([]string{link, job.title, job.company, job.location, job.summary, job.salary})
		checkErr(err)
	}
}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func checkResp(res *http.Response) {
	if res.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
	}
}
