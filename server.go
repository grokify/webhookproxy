package glipwebhookproxy

import (
	"log"

	"github.com/buaazp/fasthttprouter"
	"github.com/valyala/fasthttp"

	"github.com/grokify/glip-go-webhook"
	"github.com/grokify/glip-webhook-proxy/adapters/travisci"
	"github.com/grokify/glip-webhook-proxy/config"
)

const (
	ROUTE_SLACK_IN_GLIP           = "/webhook/slack/in/glip/:glipguid"
	ROUTE_SLACK_IN_GLIP_SLASH     = "/webhook/slack/in/glip/:glipguid/"
	ROUTE_TRAVISCI_OUT_GLIP       = "/webhook/travisci/out/glip/:glipguid"
	ROUTE_TRAVISCI_OUT_GLIP_SLASH = "/webhook/travisci/out/glip/:glipguid/"
)

func StartServer(cfg config.Configuration) {
	glip, _ := glipwebhook.NewGlipWebhookClient("")

	router := fasthttprouter.New()

	router.GET("/", HomeHandler)

	s2gHandler := NewSlackToGlipHandler(cfg, glip)
	router.POST(ROUTE_SLACK_IN_GLIP, s2gHandler.HandleFastHTTP)
	router.POST(ROUTE_SLACK_IN_GLIP_SLASH, s2gHandler.HandleFastHTTP)

	travisci2glipHandler := travisci.NewTravisciOutToGlipHandler(cfg, glip)
	router.POST(ROUTE_TRAVISCI_OUT_GLIP, travisci2glipHandler.HandleFastHTTP)
	router.POST(ROUTE_TRAVISCI_OUT_GLIP_SLASH, travisci2glipHandler.HandleFastHTTP)

	log.Fatal(fasthttp.ListenAndServe(cfg.Address(), router.Handler))
}

/*
type Configuration struct {
	Port           int
	EmojiURLFormat string
}

func (config *Configuration) Address() string {
	return fmt.Sprintf(":%d", config.Port)
}
*/
