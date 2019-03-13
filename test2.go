package main

import (
	"fmt"
	"log"
	"encoding/json"
//	"github.com/cyan21/versioning"
)

//type NPMSemVer = versioning.SemVer

type Result struct {
  Path string
//  Props versioning.SemVer
}
/*
func (sv *Result) UnmarshalJSON(b []byte) error {
  var v Result

  if err := json.Unmarshal(b, &v); err != nil {
     return err
  }

//  fmt.Println("poup")
//  sv.Path = "pout" 

  return nil
}
*/

func main() {

  var pkgs []Result

//	res := `[{"path":"/truc/chemin", "props":{"npm.version": ["0.0.1-alpha"], "npm.name": ["custom-picture"]}},{"path":"/titi/batman", "props":{"npm.version": ["0.12.0-alpha"], "npm.name": ["custom-picture"]}},{"path":"/titi/batman", "props":{"npm.version": ["0.2.0-beta"], "npm.name": ["custom-picture"]}}, {"path":"/truc/chemin2", "props":{"npm.version": ["0.0.1-release"], "npm.name": ["custom-picture"]}}, {"path":"/truc/chemin2", "props":{"npm.version": ["0.0.1-beta"], "npm.name": ["custom-gallery"]}}, {"path":"/truc/chemin2", "props":{"npm.version": ["0.0.1-release"], "npm.name": ["custom-gallery"]}}]`

//	res := `[{"path":"/truc/chemin", "props":{"major": 0, "minor": 1, "patch": 2, "maturity": "alpha"}}]`

	res := `[{"path":"/truc/chemin"},{"path":"/tto/chemin2"}]`

  if err := json.Unmarshal([]byte(res), &pkgs); err != nil {
    log.Fatal(err)
  }

  for _, v := range pkgs {
    fmt.Println(v)
  } 
}
