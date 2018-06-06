package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/google/go-querystring/query"
	"github.com/grokify/gotilla/fmt/fmtutil"

	"github.com/grokify/chathooks/src/config"
	"github.com/grokify/chathooks/src/models"

	"github.com/grokify/gotilla/io/ioutilmore"
	"github.com/grokify/gotilla/net/httputilmore"
	"github.com/grokify/gotilla/strings/stringsutil"
	"github.com/valyala/fasthttp"

	"github.com/joho/godotenv"
)

const (
	EnvWebhookUrlGlip         = "GLIP_WEBHOOK"
	EnvWebhookUrlSlack        = "SLACK_WEBHOOK"
	EnvChathooksReqInputType  = "CHATHOOKS_REQ_INPUT_TYPE"
	EnvChathooksReqOutputType = "CHATHOOKS_REQ_OUTPUT_TYPE"
	EnvChathooksReqToken      = "CHATHOOKS_REQ_TOKEN"
	EnvChathooksReqURL        = "CHATHOOKS_REQ_URL"
)

type ExampleWebhookSender struct {
	DocHandlersDir string
	BaseUrl        string
	RequestParams  models.RequestParams
}

func (s *ExampleWebhookSender) SendExamplesForInputType(inputType string) error {
	rx := regexp.MustCompile(`^event-example_.+\.(json|txt)$`)
	inputTypeDir := path.Join(s.DocHandlersDir, inputType)
	files, err := ioutilmore.DirEntriesReSizeGt0(inputTypeDir, rx)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return errors.New(fmt.Sprintf("no ^event-example_ files found for %v", inputTypeDir))
	}
	for _, file := range files {
		filepath := path.Join(inputTypeDir, file.Name())
		err := s.SendExampleForFilepath(filepath, inputType)
		if err != nil {
			return err
		}
	}
	return nil
}

func BuildURLQueryString(baseUrl string, qry interface{}) string {
	v, _ := query.Values(qry)
	qryString := v.Encode()
	if len(qryString) > 0 {
		return baseUrl + "?" + qryString
	}
	return baseUrl
}

func (s *ExampleWebhookSender) SendExampleForFilepath(filepath string, inputType string) error {
	bytes, err := ioutil.ReadFile(filepath)
	if err != nil {
		return err
	}

	qry := models.RequestParams{
		InputType:  inputType,
		OutputType: s.RequestParams.OutputType,
		Token:      s.RequestParams.Token,
		URL:        s.RequestParams.URL,
	}

	fullUrl := BuildURLQueryString(s.BaseUrl, qry)
	fmt.Printf("FULL_URL: %v\n", fullUrl)

	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()

	req.SetBody(bytes)
	req.Header.SetRequestURI(fullUrl)
	req.Header.SetMethod("POST")
	req.Header.Set(httputilmore.HeaderContentType, httputilmore.ContentTypeAppJsonUtf8)

	fastClient := fasthttp.Client{}

	err = fastClient.Do(req, resp)
	fmt.Printf("RES_STATUS: %v\n", resp.StatusCode())
	if resp.StatusCode() >= 300 || 1 == 1 {
		fmt.Printf("RES_BODY: %v\n", string(resp.Body()))
	}
	fasthttp.ReleaseRequest(req)
	fasthttp.ReleaseResponse(resp)
	return err
}

func main() {
	inputTypeP := flag.String("inputType", "travisci", "Example message type")
	urlP := flag.String("url", "https://hooks.glip.com/webhook/11112222-3333-4444-5555-666677778888", "Your Webhook URL")
	outputTypeP := flag.String("outputType", "glip", "Adapter name")
	tokenP := flag.String("token", "token", "You token")

	flag.Parse()

	qry := models.RequestParams{
		InputType:  *inputTypeP,
		OutputType: *outputTypeP,
		Token:      *tokenP,
		URL:        *urlP,
	}
	fmtutil.PrintJSON(qry)

	if len(os.Getenv("ENV_PATH")) > 0 {
		err := godotenv.Load(os.Getenv("ENV_PATH"))
		if err != nil {
			panic(err)
		}

		if len(os.Getenv(EnvChathooksReqInputType)) > 0 {
			qry.InputType = os.Getenv(EnvChathooksReqInputType)
		}
		if len(os.Getenv(EnvChathooksReqOutputType)) > 0 {
			qry.OutputType = os.Getenv(EnvChathooksReqOutputType)
		}
		if len(os.Getenv(EnvChathooksReqToken)) > 0 {
			qry.Token = os.Getenv(EnvChathooksReqToken)
		}
		if len(os.Getenv(EnvChathooksReqURL)) > 0 {
			qry.URL = os.Getenv(EnvChathooksReqURL)
		}
	}

	fmtutil.PrintJSON(qry)

	sender := ExampleWebhookSender{
		DocHandlersDir: config.DocsHandlersDir(),
		BaseUrl:        "http://localhost:8080/hook",
		RequestParams:  qry,
	}
	if len(sender.RequestParams.URL) == 0 {
		sender.RequestParams.URL = os.Getenv(EnvWebhookUrlGlip)
	}

	examples := stringsutil.SplitCondenseSpace(qry.InputType, ",")

	for _, ex := range examples {
		err := sender.SendExamplesForInputType(strings.ToLower(ex))
		if err != nil {
			panic(err)
		}
	}

	fmt.Println("DONE")
}
