package sicher

import (
    "appengine"
    "appengine/datastore"
    "appengine/taskqueue"
    "appengine/urlfetch"
    "fmt"
    "io/ioutil"
    "net/http"
    "net/url"
    "time"
)

type Site struct {
    Url       string
    CreatedAt time.Time
}

func init() {
    http.HandleFunc("/", handler)
    http.HandleFunc("/checks", checksHandler)
    http.HandleFunc("/hping", hpingHandler)
    http.HandleFunc("/notification/slack", slackHandler)
    http.HandleFunc("/sites", sitesHandler)
}

func handler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprint(w, "Hello, world!")
}

func sitesHandler(w http.ResponseWriter, r *http.Request) {
    if (r.Method == "POST") {
        createSitesHandler(w, r)
        return
    }
}

func createSitesHandler(w http.ResponseWriter, r *http.Request) {
    url := r.FormValue("url")

    c := appengine.NewContext(r)
    s := Site{
        Url: url,
        CreatedAt: time.Now(),
    }
    _, err := datastore.Put(c, datastore.NewIncompleteKey(c, "site", nil), &s)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
}

func checksHandler(w http.ResponseWriter, r *http.Request) {
    c := appengine.NewContext(r)
    q := datastore.NewQuery("site").
        Order("CreatedAt")
    var sites []Site
    _, err := q.GetAll(c, &sites)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    for _, site := range sites {
       t := taskqueue.NewPOSTTask("/hping",
         map[string][]string{"url": {site.Url}})
       if _, err := taskqueue.Add(c, t, ""); err != nil {
           http.Error(w, err.Error(), http.StatusInternalServerError)
           return
       }
    }
}

func hpingHandler(w http.ResponseWriter, r *http.Request) {
    url := r.FormValue("url")

    c := appengine.NewContext(r)
    client := urlfetch.Client(c)

    c.Debugf("HEAD " + url)
    resp, err := client.Head(url)
    if err != nil {
        c.Debugf("Fail",)
       fmt.Fprint(w, err.Error())
       // TODO datastroeに記録する
       t := taskqueue.NewPOSTTask("/notification/slack",
       map[string][]string{"url": {url}})
       if _, err := taskqueue.Add(c, t, ""); err != nil {
           http.Error(w, err.Error(), http.StatusInternalServerError)
           return
       }
       // TODO 状態を失敗に変更
    } else {
        c.Debugf("Success",)
        fmt.Fprint(w, resp.Status)
       // TODO datastroeに記録する
       // TODO 高度なチェックを起動
       // TODO 状態が失敗なら成功になったことを通知
       // TODO 状態を成功に変更
    }
}

func slackHandler(w http.ResponseWriter, r *http.Request) {
    myurl := r.FormValue("url")

    c := appengine.NewContext(r)
    token := "xoxp-XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"
    slackUrl := "https://slack.com/api/chat.postMessage"
    values := url.Values{
      "token":{token},
      "username":{"sicher"},
      "text":{"hping:" + myurl + " fail."},
      "channel":{"#general"},
    }
   client := urlfetch.Client(c)
   resp ,err  := client.PostForm(slackUrl, values)
    if err != nil {
         c.Infof("slackHandler: {err.Error()}",)
    } else {
       contents, _ := ioutil.ReadAll(resp.Body)
       fmt.Fprint(w, string(contents))
    }
}
