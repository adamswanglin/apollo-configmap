/*
 Copyright 2025 adamswanglin

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package apollosync

import (
	"encoding/json"
	"fmt"
	"github.com/iancoleman/orderedmap"
	"testing"
)

// KeyValue 用于保存键值对
type KeyValue struct {
	Key   string
	Value interface{}
}

func TestKVJson(t *testing.T) {
	// 原始 JSON
	jsonStr := `{"a": 1, "b": 2, "c": 3}`

	// 使用 orderedmap 解析
	omap := orderedmap.New()
	err := json.Unmarshal([]byte(jsonStr), omap)
	if err != nil {
		panic(err)
	}

	// 按插入顺序遍历
	for _, key := range omap.Keys() {
		value, _ := omap.Get(key)
		//convert to string

		fmt.Printf("%s: %v\n", key, value)
	}
}
