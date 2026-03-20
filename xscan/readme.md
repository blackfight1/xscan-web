## XSScan 介绍

xscan介绍和问题合集：https://ocjgibgqtg.feishu.cn/docx/JRdNd3Afqoi3yHxOowAcN7vRnQc

1. 使用html与javascript词法分析技术自动寻找反射XSS漏洞。
2. 自带爬虫(静态爬虫)，只需要输入url就能自动寻找xss漏洞。
3. 支持对get、uri、header、cookie、post检测。

### 细节
- similarity 相似度阈值灵活运用，若发现网站基本一样，相似度可以调高0.98~0.99,其他默认0.95就很高了
- 想要了解返回包细节，可将response设置为True
- 扫描完毕后通知，将finish_notify设置为True
- 不限范围扫XSS，将no_scope设置为True

## 更新记录
### 3.6.2
- [x] 可选产生多余文件删除
- [x] xss隐藏参数增强10倍
- [x] 修复解析js regex的语义错误
- [x] 添加多网页cookie支持
- [x] spider黑名单不再是正则
- [x] 支持xml xss、优化gbk编码下的xss识别
- [x] 支持agent扫描

### 3.5
- [x] cookie中xss支持
- [x] 新增启发式js隐藏参数发现
- [x] url编码，html编码 深入检查(之前只报告)
- [x] 对多参数xss结果优化
- [x] 对js template结果优化 更多token识别
- [x] 针对特殊tag title noscript textarea等的处理

### 3.0
- 优化gau结果，添加新的数据源
- 爬虫会对js分析获取更多url
- 新增被动代理功能
- 优化敏感信息收集功能
- 扫描结果分别保存
  1. spider内容 [timestamp]_spider.[ext],ext为txt或json
  2. gau内容 [timestamp]_gau.txt
  3. xss结果 [timestamp]_xss.md
  4. 敏感信息结果 [timestamp]_sensitive_info.md

### 2.5
- 更新gau爬虫，修复失效源
- 基于gau的参数分析功能更加强大

### 2.3
- 新增爬虫导出txt选项,可配合xdomscan一起使用 -output-spider-txt spider.txt

### 2.1
- 新增爬虫导出选项 -output-spider spider.jsonl
- crlf自动探测

### 1.9.3

- 报告结果增加了时间
- 增加参数 -output-json -output-md，手动指定输出文件
- 爬虫可使用表达式引擎检测敏感信息, (api文档还没整理，仿xray poc结构,可参考demo文件自行编写)
- 爬虫增加目录爆破功能，可自定义字典