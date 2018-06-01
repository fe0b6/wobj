package wobj

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	maxArgLeg  = 100000 // Максимальная длина аргумента передаваемого в js
	nodeScript = "/usr/bin/nodejs"
)

// Param это переменные для инициализации класса
type Param struct {
	Port          int
	Route         func(*Obj)
	WsRoute       func(*http.Request, *WsConn)
	WsPath        string
	MaxArgLeg     int
	YateScript    string
	NodeScript    string
	Cookie        Cookie
	CspMap        map[string]string
	Csp           string
	ParseRequest  func(http.ResponseWriter, *http.Request)
	PerfomanceLog string
}

// Cookie - Объект с описание кукисов
type Cookie struct {
	Name   string
	Domain string
	Path   string
	Time   int
	Secure bool
}

// Obj основной объект запроса
type Obj struct {
	W            http.ResponseWriter
	R            *http.Request
	TimeStart    time.Time
	Ans          Answer
	AppendFunc   func(*Obj, map[string]interface{}) map[string]interface{}
	Cache        map[string]interface{}
	Debug        bool
	ServerTiming bool
}

// Answer объект содержащий ответ
type Answer struct {
	Path     []string
	Redirect string
	Cookie   string
	Data     interface{}
	Exited   bool
	Code     int
	Meta     AnswerMeta
	IsJSON   bool
	CspMap   map[string]string
}

// AnswerMeta объект содержит заголовок и описание страницы
type AnswerMeta struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type perfomanceData struct {
	Path     string `json:"path"`
	Duration int64  `json:"duration"`
}

// WsData - структура данны хдля веб-сокета
type WsData struct {
	Action string          `json:"action"`
	Data   json.RawMessage `json:"data"`
}

// WsConn - объект коннекта вебсокета
type WsConn struct {
	Reader chan []byte
	Writer chan []byte
	Close  chan bool
	Conn   *websocket.Conn
}
