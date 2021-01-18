package sidecar

import (
	"encoding/json"
)

type MaestroRequest struct {
	Text           string `json:"text" bson:"text"`
	TextFormat     string `json:"text_format" bson:"text_format"`
	SourceLanguage string `json:"source_language" bson:"source_language"`
	TargetLanguage string `json:"target_language" bson:"target_language"`
	UID            string `json:"uid"`
}

func (r MaestroRequest) ToJSON() []byte {
	out, _ := json.Marshal(r)
	return out
}
