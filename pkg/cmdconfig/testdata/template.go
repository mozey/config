// Code generated with https://github.com/mozey/config DO NOT EDIT

package config

import (
	"bytes"
	"text/template"
)

// ExecTemplateFiz fills APP_TEMPLATE_FIZ with the given params
func (c *Config) ExecTemplateFiz(meh string) string {
	t := template.Must(template.New("templateFiz").Parse(c.templateFiz))
	b := bytes.Buffer{}
	_ = t.Execute(&b, map[string]interface{}{

		"Buz": c.buz,
		"Meh": meh,
	})
	return b.String()
}
