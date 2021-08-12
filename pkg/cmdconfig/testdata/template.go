package config

import (
	"bytes"
	"html/template"
)

// ExecTemplateFiz fills APP_TEMPLATE_FIZ with given params
func (c *Config) ExecTemplateFiz() string {
	t := template.Must(template.New("templateFiz").Parse(c.templateFiz))
	b := bytes.Buffer{}
	_ = t.Execute(&b, map[string]interface{}{
		"Buz": c.buz,
	})
	return b.String()
}
