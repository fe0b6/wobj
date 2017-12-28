package wobj

import (
	"log"
	"net/http"
	"strconv"
	"time"
)

// Начитаем слушать порт
func listen(port int) {

	http.HandleFunc("/", parseRequest)

	log.Fatalln("[fatal]", http.ListenAndServe(":"+strconv.Itoa(port), nil))
}

// Разбираем запрос
func parseRequest(w http.ResponseWriter, r *http.Request) {

	// Если сервер завершает работу
	if exited {
		w.WriteHeader(503)
		w.Write([]byte(http.StatusText(503)))
		return
	}

	// Отмечаем что начался новый запрос
	wg.Add(1)
	// По завершению запроса отмечаем что он закончился
	defer wg.Done()

	o := &Obj{R: r, W: w, TimeStart: time.Now()}
	params.Route(o)
}

// SendAnswer - функция отправки ответа
func (wo *Obj) SendAnswer() {
	// Если ничего не надо делать
	if wo.Ans.Exited {
		return
	}

	// Если нужно вернуть код
	if wo.Ans.Code != 0 {
		wo.sendCode()
		return
	}

	// Добавляем куку если надо
	if wo.Ans.Cookie != "" {
		cookie := http.Cookie{
			Name:     params.Cookie.Name,
			Domain:   params.Cookie.Domain,
			Path:     params.Cookie.Path,
			Value:    wo.Ans.Cookie,
			MaxAge:   params.Cookie.Time,
			HttpOnly: true,
			Secure:   params.Cookie.Secure,
		}
		http.SetCookie(wo.W, &cookie)
	}

	// Если переадресация
	if wo.Ans.Redirect != "" {
		wo.W.Header().Add("Expires", "Thu, 01 Jan 1970 00:00:01 GMT")
		http.Redirect(wo.W, wo.R, wo.Ans.Redirect, 301)
		return
	}

	// Формируем ответ
	str, err := wo.Tmpl()
	if err != nil {
		log.Println("[error]", err)
		wo.Ans.Code = 500
		wo.sendCode()
		return
	}

	// Добавляем csp
	if params.Csp != "" {
		wo.W.Header().Add("Content-Security-Policy", params.Csp)
	}

	wo.W.Write([]byte(str))
}

// Отправляем ответ
func (wo *Obj) sendCode() {
	wo.W.WriteHeader(wo.Ans.Code)
	// Если не 200 то добавляем статус ответа
	if wo.Ans.Code != 200 {
		wo.W.Write([]byte(http.StatusText(wo.Ans.Code)))
	}
}
