package wobj

import (
	"log"
	"os"
	"strings"
	"sync"
)

var (
	wg             sync.WaitGroup
	exited         bool
	params         Param
	wsChan         chan bool
	perfomanceFh   *os.File
	perfomanceLock sync.Mutex
	alwaysJSON     bool
)

// Init это функция инициализации
func Init(p Param) (exitChan chan bool) {
	// Устанавливаем параметры по умолчанию
	setDefaultParams(p)

	// Собираем Content-Security-Policy
	setCsp()

	// Канал для оповещения о выходе
	exitChan = make(chan bool)
	wsChan = make(chan bool)

	go waitExit(exitChan)

	if p.PerfomanceLog != "" {
		var err error
		perfomanceFh, err = os.OpenFile(p.PerfomanceLog, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0640)
		if err != nil {
			log.Println("[error]", err)
		}
	}

	// Начинаем слушать http-порт
	go listen(params.Port)

	return
}

// Ждем сигнал о выходе
func waitExit(exitChan chan bool) {
	_ = <-exitChan

	exited = true
	close(wsChan)

	log.Println("[info]", "Завершаем работу web сервера")

	// Ждем пока все запросы завершатся
	wg.Wait()

	log.Println("[info]", "Работа web сервера завершена корректно")
	exitChan <- true
}

// Устанавливаем параметры по умолчанию
func setDefaultParams(p Param) {
	params = p

	if params.MaxArgLeg == 0 {
		params.MaxArgLeg = maxArgLeg
	}

	if params.NodeScript == "" {
		params.NodeScript = nodeScript
	}

	if params.Cookie.Path == "" {
		params.Cookie.Path = "/"
	}

	if params.AlwaysJSON {
		alwaysJSON = true
	}
}

// Собираем Content-Security-Policy
func setCsp() {
	if params.CspMap == nil {
		return
	}

	csp := []string{}
	for k, v := range params.CspMap {
		csp = append(csp, k+" "+v)
	}

	params.Csp = strings.Join(csp, "; ")
}

func getCsp(h map[string]string) string {
	if params.CspMap == nil {
		return ""
	}

	csp := []string{}
	for k, v := range params.CspMap {
		if _, ok := h[k]; ok {
			v = v + " " + h[k]
		}
		csp = append(csp, k+" "+v)
	}

	return strings.Join(csp, "; ")
}

// CheckExit - Проверяем надо ли выходить
func CheckExit() bool {
	return exited
}

// StartRq - Отмечаем что идет запрос
func StartRq(i int) {
	wg.Add(i)
}

// EndRq - отмечаем что запрос закончился
func EndRq() {
	wg.Done()
}
