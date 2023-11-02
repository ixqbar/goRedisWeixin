package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/karlseguin/jsonwriter"
	"github.com/tidwall/gjson"
	"os"
	"sync"
	"time"
	"weixin/common"
)

type WValues struct {
	expireAt time.Time
	value    string
}

type WResponse struct {
	ErrorCode   int    `json:"errcode"`
	ErrorMsg    string `json:"errmsg"`
	Ticket      string `json:"ticket"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

type WItem struct {
	AppId         string
	AppSecret     string
	IsEnterprise  bool
	UseCacheFirst bool
}

type Weixin struct {
	sync.Mutex
	tokens  map[string]*WValues
	tickets map[string]*WValues
}

var wx *Weixin

func init() {
	wx = &Weixin{
		tokens:  make(map[string]*WValues, 0),
		tickets: make(map[string]*WValues, 0),
	}
}

func SaveAll() {
	wx.Lock()
	defer wx.Unlock()

	wx.save()
}

func GetToken(name string, cacheFirst bool) (*WValues, error) {
	appId := common.Config.IniCfg.Section(name).Key("app_id").String()
	appSecret := common.Config.IniCfg.Section(name).Key("app_secret").String()

	if len(appId) == 0 || len(appSecret) == 0 {
		return nil, fmt.Errorf("ERR not found match gzh config with %v", name)
	}

	isEnterprise, err := common.Config.IniCfg.Section(name).Key("is_enterprise").Bool()
	if err != nil {
		return nil, err
	}

	wi := &WItem{AppId: appId, AppSecret: appSecret, IsEnterprise: isEnterprise, UseCacheFirst: cacheFirst}

	return wx.getToken(wi, true)
}

func GetTicket(name string, cacheFirst bool) (*WValues, error) {
	isEnterprise, err := common.Config.IniCfg.Section(name).Key("is_enterprise").Bool()
	if err != nil {
		return nil, err
	}

	appId := common.Config.IniCfg.Section(name).Key("app_id").String()
	appSecret := common.Config.IniCfg.Section(name).Key("app_secret").String()

	if len(appId) == 0 || len(appSecret) == 0 {
		return nil, fmt.Errorf("ERR not found match gzh config with %v", name)
	}

	wi := &WItem{
		AppId:         appId,
		AppSecret:     appSecret,
		IsEnterprise:  isEnterprise,
		UseCacheFirst: cacheFirst,
	}

	return wx.getTicket(wi, true)
}

func (w *Weixin) getToken(wi *WItem, autoLock bool) (*WValues, error) {
	if autoLock {
		w.Lock()
		defer w.Unlock()
	}

	if wi.UseCacheFirst {
		if v, ok := w.tokens[wi.AppId]; ok && v.expireAt.After(time.Now()) {
			return v, nil
		}
	}

	/**
	非企业版
	https://developers.weixin.qq.com/doc/offiaccount/Basic_Information/Get_access_token.html
	企业版
	https://work.weixin.qq.com/api/doc/90000/90135/91039
	*/

	tokenApiUrl := ""
	if wi.IsEnterprise {
		tokenApiUrl = fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/gettoken?corpid=%s&corpsecret=%s", wi.AppId, wi.AppSecret)
	} else {
		tokenApiUrl = fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=%s&secret=%s", wi.AppId, wi.AppSecret)
	}

	res, err := Get(tokenApiUrl).Bytes()
	if err != nil {
		delete(wx.tickets, wi.AppId)
		common.Logger.Printf("request weixin token api fail appId=%s,api=%s,%v", wi.AppId, tokenApiUrl, err)
		return nil, errors.New("request weixin token api fail")
	}

	var wRes WResponse

	err = json.Unmarshal(res, &wRes)
	if err != nil || len(wRes.AccessToken) == 0 {
		delete(wx.tickets, wi.AppId)
		common.Logger.Printf("parse weixin token api response fail appId=%s,api=%s,response=%s", wi.AppId, tokenApiUrl, string(res))
		return nil, errors.New("parse weixin token api response fail")
	}

	w.tokens[wi.AppId] = &WValues{
		expireAt: time.Now().Add(time.Second * time.Duration(wRes.ExpiresIn-10)),
		value:    wRes.AccessToken,
	}

	common.Logger.Printf("refresh weixin token success appId=%s,token=%s,expireAt=%s", wi.AppId, wRes.AccessToken, w.tokens[wi.AppId].expireAt.Format("2006-01-02 15:04:05"))

	if autoLock {
		w.save()
	}

	return w.tokens[wi.AppId], nil
}

func (w *Weixin) getTicket(wi *WItem, autoLock bool) (*WValues, error) {
	if autoLock {
		w.Lock()
		defer w.Unlock()
	}

	if wi.UseCacheFirst {
		if v, ok := w.tickets[wi.AppId]; ok && v.expireAt.After(time.Now()) {
			return v, nil
		}
	}

	wxValue, err := w.getToken(wi, false)
	if err != nil {
		return nil, err
	}

	/**
	https://developers.weixin.qq.com/doc/offiaccount/OA_Web_Apps/JS-SDK.html
	企业版
	https://qyapi.weixin.qq.com/cgi-bin/get_jsapi_ticket?access_token
	*/
	ticketApiUrl := ""
	if wi.IsEnterprise {
		ticketApiUrl = fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/get_jsapi_ticket?access_token=%s", wxValue.value)
	} else {
		ticketApiUrl = fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/ticket/getticket?access_token=%s&type=jsapi", wxValue.value)
	}

	res, err := Get(ticketApiUrl).Bytes()
	if err != nil {
		common.Logger.Printf("request weixin ticket api fail appId=%s,api=%s,%v", wi.AppId, ticketApiUrl, err)
		return nil, errors.New("request weixin ticket api fail")
	}

	var wRes WResponse

	err = json.Unmarshal(res, &wRes)
	if err != nil || len(wRes.Ticket) == 0 {
		common.Logger.Printf("parse weixin ticket api response fail appId=%s,api=%s,response=%s", wi.AppId, ticketApiUrl, string(res))
		if wRes.ErrorCode == 40001 {
			if wi.UseCacheFirst {
				wi.UseCacheFirst = false
				common.Logger.Printf("will retry getTicket with no cache & lock")
				return w.getTicket(wi, false)
			} else {
				delete(w.tokens, wi.AppId)
			}
		}
		return nil, errors.New("parse weixin ticket api response fail")
	}

	w.tickets[wi.AppId] = &WValues{
		expireAt: time.Now().Add(time.Second * time.Duration(wRes.ExpiresIn-10)),
		value:    wRes.Ticket,
	}

	common.Logger.Printf("refresh weixin ticket success appId=%s,ticket=%s,expireAt=%s", wi.AppId, wRes.Ticket, w.tokens[wi.AppId].expireAt.Format("2006-01-02 15:04:05"))

	w.save()

	return w.tickets[wi.AppId], nil
}

func (w *Weixin) LoadData() {
	w.Lock()
	defer w.Unlock()

	if len(common.Config.DataFile) == 0 {
		common.Logger.Print("not found data file")
		return
	}

	jsonContent, err := os.ReadFile(common.Config.DataFile)
	if err != nil {
		common.Logger.Print(err)
		return
	}

	if !gjson.ValidBytes(jsonContent) {
		common.Logger.Printf("read error json format data")
		return
	}

	jsonResult := gjson.ParseBytes(jsonContent)

	tokens := jsonResult.Get("tokens")
	tokens.ForEach(func(key, value gjson.Result) bool {
		expireAt := time.Unix(value.Get("expireAt").Int(), 0)
		if expireAt.Before(time.Now()) {
			return true
		}

		common.Logger.Printf("iterate token,appId=%s,token=%s,expireAt=%s", key.String(), value.Get("token").String(), expireAt.String())

		wx.tokens[key.String()] = &WValues{
			expireAt: expireAt,
			value:    value.Get("token").String(),
		}

		return true
	})

	tickets := jsonResult.Get("tickets")
	tickets.ForEach(func(key, value gjson.Result) bool {
		expireAt := time.Unix(value.Get("expireAt").Int(), 0)
		if expireAt.Before(time.Now()) {
			return true
		}

		common.Logger.Printf("iterate ticket,appId=%s,ticket=%s,expireAt=%s", key.String(), value.Get("ticket").String(), expireAt.String())

		wx.tickets[key.String()] = &WValues{
			expireAt: expireAt,
			value:    value.Get("ticket").String(),
		}

		return true
	})
}

func (w *Weixin) save() {
	buffer := new(bytes.Buffer)
	jWriter := jsonwriter.New(buffer)
	jWriter.RootObject(func() {
		jWriter.KeyValue("time", time.Now().Unix())
		jWriter.Object("tokens", func() {
			for k, v := range w.tokens {
				if v.expireAt.Before(time.Now()) {
					continue
				}
				jWriter.Object(k, func() {
					jWriter.KeyValue("expireAt", v.expireAt.Unix())
					jWriter.KeyValue("token", v.value)
				})
			}
		})
		jWriter.Object("tickets", func() {
			for k, v := range w.tickets {
				if v.expireAt.Before(time.Now()) {
					continue
				}
				jWriter.Object(k, func() {
					jWriter.KeyValue("expireAt", v.expireAt.Unix())
					jWriter.KeyValue("ticket", v.value)
				})
			}
		})
	})

	err := os.WriteFile(common.Config.DataFile, buffer.Bytes(), 0644)
	if err == nil {
		common.Logger.Print("save data success")
	} else {
		common.Logger.Print(err)
	}
}
