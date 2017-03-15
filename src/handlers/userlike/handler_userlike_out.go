package userlike

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	log "github.com/Sirupsen/logrus"

	cc "github.com/commonchat/commonchat-go"
	"github.com/grokify/webhook-proxy-go/src/adapters"
	"github.com/grokify/webhook-proxy-go/src/config"
	"github.com/grokify/webhook-proxy-go/src/util"
	"github.com/valyala/fasthttp"
)

const (
	DisplayName = "Userlike"
	HandlerKey  = "userlike"
	IconURL     = "https://a.slack-edge.com/ae7f/img/services/userlike_512.png"
)

var (
	ChatMetaEvents = []string{"feedback", "forward", "rating", "receive", "start", "survey"}
	OperatorEvents = []string{"away", "back", "offline", "online"}
)

// FastHttp request handler for Userlike outbound webhook
type Handler struct {
	Config  config.Configuration
	Adapter adapters.Adapter
}

// FastHttp request handler constructor for Userlike outbound webhook
func NewHandler(cfg config.Configuration, adapter adapters.Adapter) Handler {
	return Handler{Config: cfg, Adapter: adapter}
}

// HandleFastHTTP is the method to respond to a fasthttp request.
func (h *Handler) HandleFastHTTP(ctx *fasthttp.RequestCtx) {
	ccMsg, err := Normalize(ctx.PostBody())

	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusNotAcceptable)
		log.WithFields(log.Fields{
			"type":   "http.response",
			"status": fasthttp.StatusNotAcceptable,
		}).Info(fmt.Sprintf("%v request is not acceptable.", DisplayName))
		return
	}

	util.SendWebhook(ctx, h.Adapter, ccMsg)
}

func Normalize(bodyBytes []byte) (cc.Message, error) {
	srcMsgBase, err := UserlikeBaseOutMessageFromBytes(bodyBytes)
	if err != nil {
		return cc.Message{}, err
	}
	if srcMsgBase.Type == "offline_message" && srcMsgBase.Event == "receive" {
		srcMsg, err := UserlikeOfflineMessageOutMessageFromBytes(bodyBytes)
		if err != nil {
			return cc.Message{}, err
		}
		return NormalizeOfflineMessage(srcMsg), nil
	} else if srcMsgBase.Type == "chat_meta" {
		srcMsg, err := UserlikeChatMetaStartOutMessageFromBytes(bodyBytes)
		if err != nil {
			return cc.Message{}, err
		}
		return NormalizeChatMeta(srcMsg), nil
	} else if srcMsgBase.Type == "operator" {
		srcMsg, err := UserlikeOperatorOutMessageFromBytes(bodyBytes)
		if err != nil {
			return cc.Message{}, err
		}
		return NormalizeOperator(srcMsg), nil
	} else if srcMsgBase.Type == "chat_widget" {
		srcMsg, err := UserlikeChatWidgetOutMessageFromBytes(bodyBytes)
		if err != nil {
			return cc.Message{}, err
		}
		return NormalizeChatWidget(srcMsg), nil
	}

	return cc.Message{}, errors.New("Type Not Supported")
}

func GlipActivityForChat(event string, feedback string) string {
	eventDisplay := event
	eventMap := map[string]string{
		"start":    "session started",
		"forward":  "session forwarded",
		"rating":   "rating received",
		"feedback": "feedback received",
		"survey":   "survey received",
		"receive":  "session ended",
		"goal":     "goal achieved"}
	if event == "rating" && len(feedback) > 0 {
		eventDisplay = eventMap["feedback"]
	} else {
		if displayTry, ok := eventMap[event]; ok {
			eventDisplay = displayTry
		}
	}
	return fmt.Sprintf("Chat %s", eventDisplay)
}

func NormalizeOfflineMessage(src UserlikeOfflineMessageOutMessage) cc.Message {
	message := cc.NewMessage()
	message.IconURL = IconURL

	message.Activity = fmt.Sprintf("Offline message received%v", adapters.IntegrationActivitySuffix(DisplayName))

	attachment := cc.NewAttachment()
	attachment.ThumbnailURL = IconURL

	if len(src.URL) > 0 {
		attachment.AddField(cc.Field{
			Title: "Message",
			Value: fmt.Sprintf("[%s](%v)", src.Message, src.URL)})
	}
	if len(src.ClientName) > 0 {
		attachment.AddField(cc.Field{
			Title: "Client Name",
			Value: fmt.Sprintf("%s", src.ClientName)})
	}

	if len(attachment.Fields) > 0 {
		message.AddAttachment(attachment)
	}
	return message
}

func NormalizeChatMeta(src UserlikeChatMetaStartOutMessage) cc.Message {
	message := cc.NewMessage()
	message.IconURL = IconURL
	message.Activity = fmt.Sprintf("%s%s",
		GlipActivityForChat(src.Event, src.FeedbackMessage), adapters.IntegrationActivitySuffix(DisplayName))

	attachment := cc.NewAttachment()
	attachment.ThumbnailURL = IconURL

	displayedUrl := false

	if src.Event == "rating" || src.Event == "survey" { // includes feedback
		if len(src.FeedbackMessage) > 0 {
			url, linked := LinkifyURL(src.FeedbackMessage, src.URL, displayedUrl)
			displayedUrl = linked
			attachment.AddField(cc.Field{
				Title: "Feedback",
				Value: url,
				Short: false})
		}
		if len(src.PostSurveyOption) > 0 {
			url, linked := LinkifyURL(src.PostSurveyOption, src.URL, displayedUrl)
			displayedUrl = linked
			attachment.AddField(cc.Field{
				Title: "Rating",
				Value: url,
				Short: true})
		}
	}
	if len(src.ClientName) > 0 {
		url, linked := LinkifyURL(src.ClientName, src.URL, displayedUrl)
		displayedUrl = linked
		attachment.AddField(cc.Field{
			Title: "Client Name",
			Value: url,
			Short: true})
	} else {
		url, linked := LinkifyURL("Unknown", src.URL, displayedUrl)
		displayedUrl = linked
		attachment.AddField(cc.Field{
			Title: "Client Name",
			Value: url,
			Short: true})
	}

	if len(attachment.Fields) > 0 {
		message.AddAttachment(attachment)
	}
	return message
}

func LinkifyURL(innerHtml string, url string, skipLinking bool) (string, bool) {
	//linked := false
	if len(innerHtml) == 0 && len(url) > 0 {
		innerHtml = url
	}
	if skipLinking == true {
		return innerHtml, skipLinking
	}
	if len(url) < 1 {
		return innerHtml, false
	}
	if len(innerHtml) > 0 {
		return fmt.Sprintf("[%s](%s)", innerHtml, url), true
	}
	return fmt.Sprintf("[%s](%s)", url, url), true
}

func NormalizeChatWidget(src UserlikeChatWidgetOutMessage) cc.Message {
	message := cc.NewMessage()
	message.IconURL = IconURL
	message.Activity = fmt.Sprintf("Chat widget configuration updated%s", adapters.IntegrationActivitySuffix(DisplayName))

	titleParts := []string{}
	if len(src.StatusUrl) > 0 {
		titleParts = append(titleParts, fmt.Sprintf("[Check status](%s)", src.StatusUrl))
	}
	if len(src.TestUrl) > 0 {
		titleParts = append(titleParts, fmt.Sprintf("[test widget](%s)", src.TestUrl))
	}
	if len(titleParts) > 0 {
		message.Title = strings.Join(titleParts, " and ")
	}

	attachment := cc.NewAttachment()
	attachment.ThumbnailURL = IconURL

	if len(src.Name) > 0 {
		attachment.AddField(cc.Field{
			Title: "Widget Name",
			Value: fmt.Sprintf("[%s](%s)", src.Name, src.CustomUrl),
			Short: true})
	}
	attachment.AddField(cc.Field{
		Title: "Widget Version",
		Value: fmt.Sprintf("%v", src.WidgetVersion),
		Short: true})
	if len(src.WidgetExternalType) > 0 {
		attachment.AddField(cc.Field{
			Title: "Widget Type",
			Value: fmt.Sprintf("%v", src.WidgetExternalType),
			Short: true})
	}

	message.AddAttachment(attachment)
	return message
}

func NormalizeOperator(src UserlikeOperatorOutMessage) cc.Message {
	message := cc.NewMessage()
	message.IconURL = IconURL
	message.Activity = fmt.Sprintf("Operator is %s%s",
		src.Event, adapters.IntegrationActivitySuffix(DisplayName))

	attachment := cc.NewAttachment()

	if len(src.DashboardUrl) > 0 {
		attachment.AddField(cc.Field{
			Title: "Operator",
			Value: fmt.Sprintf("[%s](%s)", src.Name, src.DashboardUrl)})
	}

	message.AddAttachment(attachment)
	return message
}

type UserlikeBaseOutMessage struct {
	Event string `json:"_event,omitempty"`
	Type  string `json:"_type,omitempty"`
}

func UserlikeBaseOutMessageFromBytes(bytes []byte) (UserlikeBaseOutMessage, error) {
	msg := UserlikeBaseOutMessage{}
	err := json.Unmarshal(bytes, &msg)
	return msg, err
}

type UserlikeOfflineMessageOutMessage struct {
	UserlikeBaseOutMessage
	BrowserName     string             `json:"browser_name,omitempty"`
	BrowserOS       string             `json:"browser_os,omitempty"`
	BrowserVersion  string             `json:"browser_version,omitempty"`
	ChatWidget      UserlikeChatWidget `json:"chat_widget,omitempty"`
	ClientEmail     string             `json:"client_email,omitempty"`
	ClientName      string             `json:"client_name,omitempty"`
	CreatedAt       string             `json:"created_at,omitempty"`
	Custom          interface{}        `json:"custom,omitempty"`
	DataPrivacy     interface{}        `json:"data_privacy,omitempty"`
	Id              int64              `json:"id,omitempty"`
	LocCity         string             `json:"loc_city,omitempty"`
	LocCountry      string             `json:"loc_country,omitempty"`
	LocLat          float64            `json:"loc_lat,omitempty"`
	LocLon          float64            `json:"loc_lon,omitempty"`
	MarkedRead      bool               `json:"marked_read,omitempty"`
	Message         string             `json:"message,omitempty"`
	PageImpressions int64              `json:"page_impresions,omitempty"`
	ScreenshotOID   string             `json:"screenshot_oid,omitempty"`
	ScreenshotURL   string             `json:"screenshot_url,omitempty"`
	Status          string             `json:"status,omitempty"`
	Topic           string             `json:"topic,omitempty"`
	URL             string             `json:"url,omitempty"`
	Visits          int64              `json:"visits,omitempty"`
}

func UserlikeOfflineMessageOutMessageFromBytes(bytes []byte) (UserlikeOfflineMessageOutMessage, error) {
	msg := UserlikeOfflineMessageOutMessage{}
	err := json.Unmarshal(bytes, &msg)
	return msg, err
}

type UserlikeChatWidget struct {
	Id   int64  `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type UserlikeChatMetaStartOutMessage struct {
	UserlikeBaseOutMessage
	BrowserName       string             `json:"browser_name,omitempty"`
	BrowserOS         string             `json:"browser_os,omitempty"`
	BrowserVersion    string             `json:"browser_version,omitempty"`
	ChatWidget        UserlikeChatWidget `json:"chat_widget,omitempty"`
	ClientEmail       string             `json:"client_email,omitempty"`
	ClientName        string             `json:"client_name,omitempty"`
	ClientUUID        string             `json:"client_uuid,omitempty"`
	CreatedAt         string             `json:"created_at,omitempty"`
	DataPrivacy       bool               `json:"data_privacy,omitempty"`
	Duration          string             `json:"duration,omitempty"`
	EndedAt           string             `json:"ended_at,omitempty"`
	FeedbackMessage   string             `json:"feedback_message,omitempty"`
	Id                int64              `json:"id,omitempty"`
	InitialURL        string             `json:"initial_url,omitempty"`
	LocCity           string             `json:"browser_version,omitempty"`
	LocCountry        string             `json:"browser_version,omitempty"`
	LocLat            float64            `json:"loc_lat,omitempty"`
	LocLon            float64            `json:"loc_lon,omitempty"`
	MarkedRead        bool               `json:"marked_read,omitempty"`
	OperatorCurrentId int64              `json:"operator_current_id,omitempty"`
	PageImpressions   int64              `json:"page_impressions,omitempty"`
	PostSurveyOption  string             `json:"post_survey_option,omitempty"`
	Rate              int64              `json:"rate,omitempty"`
	Referrer          string             `json:"referrer,omitempty"`
	Status            string             `json:"status,omitempty"`
	Topic             string             `json:"topic,omitempty"`
	URL               string             `json:"url,omitempty"`
	Visits            int64              `json:"visits,omitempty"`
	WasProactive      bool               `json:"was_proactive,omitempty"`
}

func UserlikeChatMetaStartOutMessageFromBytes(bytes []byte) (UserlikeChatMetaStartOutMessage, error) {
	msg := UserlikeChatMetaStartOutMessage{}
	err := json.Unmarshal(bytes, &msg)
	return msg, err
}

type UserlikeOperatorOutMessage struct {
	UserlikeBaseOutMessage
	DashboardUrl    string        `json:"dashboard_url,omitempty"`
	Email           string        `json:"email,omitempty"`
	FirstName       string        `json:"first_name,omitempty"`
	Id              int64         `json:"id,omitempty"`
	IsActive        bool          `json:"is_active,omitempty"`
	JID             string        `json:"jid,omitempty"`
	Lang            string        `json:"lang,omitempty"`
	LastName        string        `json:"last_name,omitempty"`
	Locale          string        `json:"locale,omitempty"`
	Name            string        `json:"name,omitempty"`
	OperatorGroup   OperatorGroup `json:"operator_group,omitempty"`
	OperatorGroupId int64         `json:"operator_group_id,omitempty"`
	Role            string        `json:"role,omitempty"`
	RoleName        string        `json:"role_name,omitempty"`
	Timezone        string        `json:"timezone,omitempty"`
	UrlImage        string        `json:"url_image,omitempty"`
	Username        string        `json:"username,omitempty"`
}

func UserlikeOperatorOutMessageFromBytes(bytes []byte) (UserlikeOperatorOutMessage, error) {
	msg := UserlikeOperatorOutMessage{}
	err := json.Unmarshal(bytes, &msg)
	return msg, err
}

type OperatorGroup struct {
	Id   int64  `json:"id,omitempty"`
	Name string `json:"string,omitempty"`
}

type UserlikeChatWidgetOutMessage struct {
	UserlikeBaseOutMessage
	CustomUrl          string `json:"custom_url,omitempty"`
	Name               string `json:"name,omitempty"`
	TransitionDuration int64  `json:"transition_duration,omitempty"`
	StatusUrl          string `json:"status_url,omitempty"`
	TestUrl            string `json:"test_url,omitempty"`
	WidgetExternalType string `json:"widget_external_type,omitempty"`
	WidgetVersion      int64  `json:"widget_version,omitempty"`
}

func UserlikeChatWidgetOutMessageFromBytes(bytes []byte) (UserlikeChatWidgetOutMessage, error) {
	msg := UserlikeChatWidgetOutMessage{}
	err := json.Unmarshal(bytes, &msg)
	return msg, err
}
