﻿package main

import (
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"strings"
)

type templateData struct {
	StringMap         map[string]string
	IntMap            map[string]int
	FloatMap          map[string]float32
	Data              map[string]interface{}
	CSRFToken         string
	Flash             string
	Warning           string
	Error             string
	IsAuthenticated   bool
	API               string
	CSSVersion        string
	StripeSecret      string
	StripePublishable string
	UserID            int
}

var functions = template.FuncMap{
	"formatCurrency": formatCurrency,
}

func formatCurrency(n int) string {
	f := float32(n) / float32(100)
	return fmt.Sprintf("$%.2f", f)
}

//go:embed templates
var templateFS embed.FS

func (app *application) addDefaultData(td *templateData, r *http.Request) *templateData {

	td.API = app.config.api
	td.StripePublishable = app.config.stripe.key
	td.StripeSecret = app.config.stripe.secret

	if app.Session.Exists(r.Context(), "userID") {
		td.IsAuthenticated = true
		td.UserID = app.Session.GetInt(r.Context(), "userID")
	}

	return td
}

func (app *application) renderTemplate(w http.ResponseWriter, r *http.Request, page string, td *templateData, partials ...string) error {
	var t *template.Template
	var err error

	templateToRender := fmt.Sprintf("templates/%s.page.gohtml", page)

	_, isTemplateInMap := app.templateCache[templateToRender]

	if isTemplateInMap {
		t = app.templateCache[templateToRender]
	} else {
		t, err = app.parseTemplate(partials, page, templateToRender)

		if err != nil {
			app.errorLog.Println(err)
			return err
		}
	}

	if td == nil {
		td = &templateData{}
	}

	td = app.addDefaultData(td, r)
	err = t.Execute(w, td)

	if err != nil {
		app.errorLog.Println(err)
		return err
	}

	return nil
}

func (app *application) parseTemplate(partials []string, page, templateToRender string) (*template.Template, error) {
	var t *template.Template
	var err error

	if len(partials) > 0 {
		for i, name := range partials {
			partials[i] = fmt.Sprintf("templates/%s.partial.gohtml", name)
		}
	}

	if len(partials) > 0 {
		t, err = template.New(fmt.Sprintf("%s.page.gohtml", page)).Funcs(functions).ParseFS(templateFS, "templates/base.layout.gohtml", strings.Join(partials, ","), templateToRender)
	} else {
		t, err = template.New(fmt.Sprintf("%s.page.gohtml", page)).Funcs(functions).ParseFS(templateFS, "templates/base.layout.gohtml", templateToRender)
	}

	if err != nil {
		app.errorLog.Println(err)
		return nil, err
	}

	app.templateCache[templateToRender] = t

	return t, nil
}
