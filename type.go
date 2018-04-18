package wobj

import (
	"net/http"
	"time"
)

const (
	maxArgLeg  = 100000 // Максимальная длина аргумента передаваемого в js
	nodeScript = "/usr/bin/nodejs"
)

// Param это переменные для инициализации класса
type Param struct {
	Port         int
	Route        func(*Obj)
	WsRoute      func(w http.ResponseWriter, r *http.Request)
	WsPath       string
	MaxArgLeg    int
	YateScript   string
	NodeScript   string
	Cookie       Cookie
	CspMap       map[string]string
	Csp          string
	ParseRequest func(http.ResponseWriter, *http.Request)
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
	W          http.ResponseWriter
	R          *http.Request
	TimeStart  time.Time
	Ans        Answer
	AppendFunc func(*Obj, map[string]interface{}) map[string]interface{}
	Cache      map[string]interface{}
	Debug      bool
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
}

// AnswerMeta объект содержит заголовок и описание страницы
type AnswerMeta struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}
