package main

import(
  "fmt"
)

type SemVer interface {
  Newer(s SemVer) int
}

type NPMSemVer struct {
  Major int
  Minor int
  Patch int
}

func (n NPMSemVer) Newer(s2 SemVer) int {
  fmt.Println("NPM comparison") 
  switch s2.(type) {
    case NugetSemVer: fmt.Println("it's a nuget semver") 
    case NPMSemVer: fmt.Println("it's a NPM semver") 
    default: fmt.Println("it's a ??? semver")
  }
  return 1
}

type NugetSemVer struct {
  Major int
  Minor int
  Patch int
  Maturity string 
}

func (n NugetSemVer) Newer(s2 SemVer) int {
  switch s2.(type) {
    case NugetSemVer: fmt.Println("it's a nuget semver") 
    case NPMSemVer: fmt.Println("it's a NPM semver") 
    default: fmt.Println("it's a ??? semver")
  }

  fmt.Println("Nuget comparison") 
  return 1
}


func main(){
  arr := make([]SemVer, 5)
  arr[0] = NPMSemVer{Major: 7, Minor:2, Patch:8}
  arr[1] = NugetSemVer{Major: 1, Minor:2, Patch:3, Maturity: "rc"}
  arr[2] = NPMSemVer{Major: 7, Minor:12, Patch:8}
  
  fmt.Println(arr)
  fmt.Println("comparison :", arr[0].Newer(arr[1]))
}
