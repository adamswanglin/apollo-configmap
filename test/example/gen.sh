#!/bin/bash

#
# Copyright 2025 adamswanglin
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

# 输出文件名
output_file="apollo_config.yaml"

# 清空或创建文件
echo -n "" > "$output_file"

# 循环生成1000份配置
for i in $(seq 1 1000); do
  cat <<EOF >> "$output_file"
apiVersion: apollo.adamswanglin.com/v1
kind: ApolloConfig
metadata:
  labels:
    apolloConfigServer: default-demo-server
  name: demo-config-$i
  namespace: default
spec:
  apollo:
    accessKeySecret: 5e4f59f2035046c2a18e53e31b138f93
    appId: "000111"
    clusterName: default
    namespaceName: application
  apolloConfigServer: default/demo-server
  configMap: apollo-config-$i
  fileName: application.properties
EOF

  # 如果不是最后一个块，添加分隔符
  if [ $i -ne 1000 ]; then
    echo -e "---" >> "$output_file"
  fi

done

echo "生成完成，结果保存在 $output_file"
