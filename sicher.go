package sicher

import (
    "appengine"
    "appengine/datastore"
    "appengine/taskqueue"
    "appengine/urlfetch"
    "appengine/user"
    "fmt"
    "io/ioutil"
    "net/http"
    "net/url"
    "strconv"
    "time"
)

type Site struct {
    Url       string
    Owners    []string
    CreatedAt time.Time
    UpdatedAt time.Time
}

func init() {
    http.HandleFunc("/", handler)
    http.HandleFunc("/backend/checks", checksHandler)
    http.HandleFunc("/backend/hping", hPingHandler)
    http.HandleFunc("/backend/notification/slack", slackHandler)
    http.HandleFunc("/sites", sitesHandler)
    http.HandleFunc("/signOut", signOutHandler)
}

func signOutHandler(w http.ResponseWriter, r *http.Request) {
    c := appengine.NewContext(r)
    u := user.Current(c)
    if u == nil {
        w.Header().Set("Location", "/")
        w.WriteHeader(http.StatusFound)
    }
    url, err := user.LogoutURL(c,r.URL.String())
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    w.Header().Set("Location", url)
    w.WriteHeader(http.StatusFound)
}

func handler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprint(w, "Hello, world!")
}

func sitesHandler(w http.ResponseWriter, r *http.Request) {
    c := appengine.NewContext(r)
    u := user.Current(c)
    switch r.Method {
    case "GET":{
        c := appengine.NewContext(r)
        fmt.Fprintln(w, u)
        if u.Admin {
            fmt.Fprintln(w, "!!admin!!")
        }
        q := datastore.NewQuery("site").
        Filter("Owners =", u.Email).
        Order("CreatedAt")
        if u.Admin {
            q = querySiteAll()
        }

        var sites []Site
        keys, err := q.GetAll(c, &sites)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        for i, site := range sites {
            fmt.Fprint(w,keys[i])
            fmt.Fprint(w,":" + site.Url + "\n")
        }
        fmt.Fprintln(w, "sicher")
    }
    case "POST": createSites(w, r, u)
    case "DELETE":
        {
            c := appengine.NewContext(r)
            intId, err := strconv.ParseInt(r.FormValue("intId"), 0, 64)
            if err != nil {
                http.Error(w, err.Error(), http.StatusInternalServerError)
                return
            }
            key := datastore.NewKey(c, "site", "", intId, nil)
            if u.Admin {
                datastore.Delete(c, key)
            }
        }
    }
}

func createSites(w http.ResponseWriter, r *http.Request, u *user.User) {
    url := r.FormValue("url")

    c := appengine.NewContext(r)
    q := datastore.NewQuery("site").Filter("Url =", url)
    var sites []Site
    keys, err := q.GetAll(c, &sites)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    var site Site
    var key *datastore.Key
    if len(sites) == 0 {
        site = Site{
            Url: url,
            Owners: []string{u.Email},
            CreatedAt: time.Now(),
            UpdatedAt: time.Now(),
        }
        key = datastore.NewIncompleteKey(c, "site", nil)
    } else {
        key = keys[0]
        site = sites[0]
        any := false
        for _, owner := range site.Owners {
            if owner == u.Email {
                any = true
            }
        }
        if !any {
            site.Owners = append(site.Owners, u.Email)
            site.UpdatedAt = time.Now()
        }
    }

    _, err2 := datastore.Put(c, key, &site)
    if err2 != nil {
        http.Error(w, err2.Error(), http.StatusInternalServerError)
        return
    }
}

func checksHandler(w http.ResponseWriter, r *http.Request) {
    c := appengine.NewContext(r)
    q := querySiteAll()
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
    fmt.Fprint(w, "Success")
}

func hPingHandler(w http.ResponseWriter, r *http.Request) {
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

func querySiteAll() *datastore.Query {
    return datastore.NewQuery("site").
        Order("CreatedAt")
}
