package main

import "fmt"

func main() {
  //a := [5]int{11,12,13,14,15}
  //v := a[0:3]
  //v[0] = 21
  
  //v := []int{2,4,6} 
  //v[2] = 777

  v := make(map[string][]int)
  //v["test"] = []int{7,8,9} 
  v["test"] = make([]int,1)
//  v["test"][0] = 5
  v["test"] = append(v["test"],10)
  v["test"] = append(v["test"],11)
  v["tutu"] = []int{77,88,99} 
  for k,i := range v {
     fmt.Println(k, i[0], i[1], i[2])
  }

}
