# 企业内部开发
企业内部开发
第三方应用开发
服务商代开发
智慧硬件开发
群机器人配置说明
最后更新：2025/06/04
如何使用群机器人
消息类型及数据格式
      文本类型
      markdown类型
      markdown_v2类型
      图片类型
      图文类型
      文件类型
      语音类型
      模版卡片类型
            文本通知模版卡片
            图文展示模版卡片
消息发送频率限制
文件上传接口
如何使用群机器人
在终端某个群组添加机器人之后，创建者可以在机器人详情页看到该机器人特有的webhookurl。开发者可以按以下说明向这个地址发起HTTP POST 请求，即可实现给该群组发送消息。下面举个简单的例子.
假设webhook是：https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=693a91f6-7xxx-4bc4-97a0-0ec2sifa5aaa
特别特别要注意：一定要保护好机器人的webhook地址，避免泄漏！不要分享到github、博客等可被公开查阅的地方，否则坏人就可以用你的机器人来发垃圾消息了。

以下是用curl工具往群组推送文本消息的示例（注意要将url替换成你的机器人webhook地址，content必须是utf8编码）：

```
curl 'https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=693axxx6-7aoc-4bc4-97a0-0ec2sifa5aaa' \
   -H 'Content-Type: application/json' \
   -d '
   {
    	"msgtype": "text",
    	"text": {
        	"content": "hello world"
    	}
   }'
```
当前自定义机器人支持文本（text）、markdown（markdown、markdown_v2）、图片（image）、图文（news）、文件（file）、语音（voice）、模板卡片（template_card）八种消息类型。
机器人的text/markdown类型消息支持在content中使用<@userid>扩展语法来@群成员（markdown_v2类型消息不支持该扩展语法）
消息类型及数据格式
## 文本类型
```json
{
    "msgtype": "text",
    "text": {
        "content": "广州今日天气：29度，大部分多云，降雨概率：60%",
		"mentioned_list":["wangqing","@all"],
		"mentioned_mobile_list":["13800001111","@all"]
    }
}
```
| 参数 | 是否必填 | 说明 |
|---|---|---|
| msgtype | 是 | 消息类型，此时固定为text |
| content | 是 | 文本内容，最长不超过2048个字节，必须是utf8编码 |
| mentioned_list | 否 | userid的列表，提醒群中的指定成员(@某个成员)，@all表示提醒所有人，如果开发者获取不到userid，可以使用mentioned_mobile_list |
| mentioned_mobile_list | 否 | 手机号列表，提醒手机号对应的群成员(@某个成员)，@all表示提醒所有人 |

## markdown类型
```json
{
    "msgtype": "markdown",
    "markdown": {
        "content": "实时新增用户反馈<font color=\"warning\">132例</font>，请相关同事注意。\n>类型:<font color=\"comment\">用户反馈</font>\n>普通用户反馈:<font color=\"comment\">117例</font>\n>VIP用户反馈:<font color=\"comment\">15例</font>"
    }
}
```
| 参数 | 是否必填 | 说明 |
|---|---|---|
| msgtype | 是 | 消息类型，此时固定为markdown |
| content | 是 | markdown内容，最长不超过4096个字节，必须是utf8编码 |

目前支持的markdown语法是如下的子集：

标题 （支持1至6级标题，注意#与文字中间要有空格）

# 标题一
## 标题二
### 标题三
#### 标题四
##### 标题五
###### 标题六
加粗

**bold**
链接

[这是一个链接](https://work.weixin.qq.com/api/doc)
行内代码段（暂不支持跨行）

`code`
引用

> 引用文字
字体颜色(只支持3种内置颜色)

<font color="info">绿色</font>
<font color="comment">灰色</font>
<font color="warning">橙红色</font>

## markdown_v2类型
```json
{
	"msgtype": "markdown_v2",
	"markdown_v2": {
         "content": "# 一、标题\n## 二级标题\n### 三级标题\n# 二、字体\n*斜体*\n\n**加粗**\n# 三、列表 \n- 无序列表 1 \n- 无序列表 2\n  - 无序列表 2.1\n  - 无序列表 2.2\n1. 有序列表 1\n2. 有序列表 2\n# 四、引用\n> 一级引用\n>>二级引用\n>>>三级引用\n# 五、链接\n[这是一个链接](https:work.weixin.qq.com\/api\/doc)\n![](https://res.mail.qq.com/node/ww/wwopenmng/images/independent/doc/test_pic_msg1.png)\n# 六、分割线\n\n---\n# 七、代码\n`这是行内代码`\n```\n这是独立代码块\n```\n\n# 八、表格\n| 姓名 | 文化衫尺寸 | 收货地址 |\n| :----- | :----: | -------: |\
| 张三 | S | 广州 |\n| 李四 | L | 深圳 |\n"
	   }
}
```
| 参数 | 是否必填 | 说明 |
|---|---|---|
| msgtype | 是 | 消息类型，此时固定为markdown_v2。 |
| content | 是 | markdown_v2内容，最长不超过4096个字节，必须是utf8编码。 |

特殊的，
1. markdown_v2不支持字体颜色、@群成员的语法， 具体支持的语法可参考下面说明
2. 消息内容在客户端 4.1.36 版本以下(安卓端为4.1.38以下) 消息表现为纯文本，建议使用最新客户端版本体验

目前支持的markdown_v2语法是如下的子集：

标题 （支持1至6级标题，注意#与文字中间要有空格）

# 标题一
## 标题二
### 标题三
#### 标题四
##### 标题五
###### 标题六
字体

*斜体*
**加粗**
列表

- 无序列表 1
- 无序列表 2
 - 无序列表 2.1
 - 无序列表 2.2
1. 有序列表 1
2. 有序列表 2
引用

>一级引用
>>二级引用
>>>三级引用
链接

[这是一个链接](https://work.weixin.qq.com/api/doc)
![这是一个图片](https://res.mail.qq.com/node/ww/wwopenmng/images/independent/doc/test_pic_msg1.png)
分割线


---
代码

`这是行内代码`

```
这是独立代码块
```



表格

| 姓名 | 文化衫尺寸 | 收货地址 |
| :----- | :----: | -------: |
| 张三 | S | 广州 |
| 李四 | L | 深圳 |

## 图片类型
```json
{
    "msgtype": "image",
    "image": {
        "base64": "DATA",
		"md5": "MD5"
    }
}
```
| 参数 | 是否必填 | 说明 |
|---|---|---|
| msgtype | 是 | 消息类型，此时固定为image |
| base64 | 是 | 图片内容的base64编码 |
| md5 | 是 | 图片内容（base64编码前）的md5值 |
注：图片（base64编码前）最大不能超过2M，支持JPG,PNG格式

## 图文类型
```json
{
    "msgtype": "news",
    "news": {
       "articles" : [
           {
               "title" : "中秋节礼品领取",
               "description" : "今年中秋节公司有豪礼相送",
               "url" : "www.qq.com",
               "picurl" : "https://res.mail.qq.com/node/ww/wwopenmng/images/independent/doc/test_pic_msg1.png"
           }
        ]
    }
}
```
| 参数 | 是否必填 | 说明 |
|---|---|---|
| msgtype | 是 | 消息类型，此时固定为news |
| articles | 是 | 图文消息，一个图文消息支持1到8条图文 |
| title | 是 | 标题，不超过128个字节，超过会自动截断 |
| description | 否 | 描述，不超过512个字节，超过会自动截断 |
| url | 是 | 点击后跳转的链接。 |
| picurl | 否 | 图文消息的图片链接，支持JPG、PNG格式，较好的效果为大图 1068*455，小图150*150。 |

## 文件类型
```json
{
    "msgtype": "file",
    "file": {
 		"media_id": "3a8asd892asd8asd"
    }
}
```
| 参数 | 是否必填 | 说明 |
|---|---|---|
| msgtype | 是 | 消息类型，此时固定为file |
| media_id | 是 | 文件id，通过下文的文件上传接口获取 |

## 语音类型
```json
{
	"msgtype": "voice",
	"voice": {
		"media_id": "MEDIA_ID"
	}
}
```
| 参数 | 是否必填 | 说明 |
|---|---|---|
| msgtype | 是 | 语音类型，此时固定为voice |
| media_id | 是 | 语音文件id，通过下文的文件上传接口获取 |

## 模版卡片类型
### 文本通知模版卡片
```json
{
    "msgtype":"template_card",
    "template_card":{
        "card_type":"text_notice",
        "source":{
            "icon_url":"https://wework.qpic.cn/wwpic/252813_jOfDHtcISzuodLa_1629280209/0",
            "desc":"企业微信",
            "desc_color":0
        },
        "main_title":{\
            "title":"欢迎使用企业微信",
            "desc":"您的好友正在邀请您加入企业微信"
        },
        "emphasis_content":{\
            "title":"100",
            "desc":"数据含义"
        },
        "quote_area":{\
            "type":1,
            "url":"https://work.weixin.qq.com/?from=openApi",
            "appid":"APPID",
            "pagepath":"PAGEPATH",
            "title":"引用文本标题",
            "quote_text":"Jack：企业微信真的很好用~\nBalian：超级好的一款软件！"
        },
        "sub_title_text":"下载企业微信还能抢红包！",
        "horizontal_content_list":[\
            {
                "keyname":"邀请人",
                "value":"张三"
            },
            {
                "keyname":"企微官网",
                "value":"点击访问",
                "type":1,
                "url":"https://work.weixin.qq.com/?from=openApi"
            },
            {
                "keyname":"企微下载",
                "value":"企业微信.apk",
                "type":2,
                "media_id":"MEDIAID"
            }
        ],
        "jump_list":[\
            {
                "type":1,
                "url":"https://work.weixin.qq.com/?from=openApi",
                "title":"企业微信官网"
            },
            {
                "type":2,
                "appid":"APPID",
                "pagepath":"PAGEPATH",
                "title":"跳转小程序"
            }
        ],
        "card_action":{\
            "type":1,
            "url":"https://work.weixin.qq.com/?from=openApi",
            "appid":"APPID",
            "pagepath":"PAGEPATH"
        }
    }
}
```

请求参数

| 参数 | 类型 | 必须 | 说明 |
|---|---|---|---|
| msgtype | String | 是 | 消息类型，此时的消息类型固定为template_card |
| template_card | Object | 是 | 具体的模版卡片参数 |

template_card的参数说明

| 参数 | 类型 | 必须 | 说明 |
|---|---|---|---|
| card_type | String | 是 | 模版卡片的模版类型，文本通知模版卡片的类型为text_notice |
| source | Object | 否 | 卡片来源样式信息，不需要来源样式可不填写 |
| source.icon_url | String | 否 | 来源图片的url |
| source.desc | String | 否 | 来源图片的描述，建议不超过13个字 |
| source.desc_color | Int | 否 | 来源文字的颜色，目前支持：0(默认) 灰色，1 黑色，2 红色，3 绿色 |
| main_title | Object | 是 | 模版卡片的主要内容，包括一级标题和标题辅助信息 |
| main_title.title | String | 否 | 一级标题，建议不超过26个字。模版卡片主要内容的一级标题main_title.title和二级普通文本sub_title_text必须有一项填写 |
| main_title.desc | String | 否 | 标题辅助信息，建议不超过30个字 |
| emphasis_content | Object | 否 | 关键数据样式 |
| emphasis_content.title | String | 否 | 关键数据样式的数据内容，建议不超过10个字 |
| emphasis_content.desc | String | 否 | 关键数据样式的数据描述内容，建议不超过15个字 |
| quote_area | Object | 否 | 引用文献样式，建议不与关键数据共用 |
| quote_area.type | Int | 否 | 引用文献样式区域点击事件，0或不填代表没有点击事件，1 代表跳转url，2 代表跳转小程序 |
| quote_area.url | String | 否 | 点击跳转的url，quote_area.type是1时必填 |
| quote_area.appid | String | 否 | 点击跳转的小程序的appid，quote_area.type是2时必填 |
| quote_area.pagepath | String | 否 | 点击跳转的小程序的pagepath，quote_area.type是2时选填 |
| quote_area.title | String | 否 | 引用文献样式的标题 |
| quote_area.quote_text | String | 否 | 引用文献样式的引用文案 |
| sub_title_text | String | 否 | 二级普通文本，建议不超过112个字。模版卡片主要内容的一级标题main_title.title和二级普通文本sub_title_text必须有一项填写 |
| horizontal_content_list | Object[] | 否 | 二级标题+文本列表，该字段可为空数组，但有数据的话需确认对应字段是否必填，列表长度不超过6 |
| horizontal_content_list.type | Int | 否 | 模版卡片的二级标题信息内容支持的类型，1是url，2是文件附件，3 代表点击跳转成员详情 |
| horizontal_content_list.keyname | String | 是 | 二级标题，建议不超过5个字 |
| horizontal_content_list.value | String | 否 | 二级文本，如果horizontal_content_list.type是2，该字段代表文件名称（要包含文件类型），建议不超过26个字 |
| horizontal_content_list.url | String | 否 | 链接跳转的url，horizontal_content_list.type是1时必填 |
| horizontal_content_list.media_id | String | 否 | 附件的media_id，horizontal_content_list.type是2时必填 |
| horizontal_content_list.userid | String | 否 | 成员详情的userid，horizontal_content_list.type是3时必填 |
| jump_list | Object[] | 否 | 跳转指引样式的列表，该字段可为空数组，但有数据的话需确认对应字段是否必填，列表长度不超过3 |
| jump_list.type | Int | 否 | 跳转链接类型，0或不填代表不是链接，1 代表跳转url，2 代表跳转小程序 |
| jump_list.title | String | 是 | 跳转链接样式的文案内容，建议不超过13个字 |
| jump_list.url | String | 否 | 跳转链接的url，jump_list.type是1时必填 |
| jump_list.appid | String | 否 | 跳转链接的小程序的appid，jump_list.type是2时必填 |
| jump_list.pagepath | String | 否 | 跳转链接的小程序的pagepath，jump_list.type是2时选填 |
| card_action | Object | 是 | 整体卡片的点击跳转事件，text_notice模版卡片中该字段为必填项 |
| card_action.type | Int | 是 | 卡片跳转类型，1 代表跳转url，2 代表打开小程序。text_notice模版卡片中该字段取值范围为[1,2] |
| card_action.url | String | 否 | 跳转事件的url，card_action.type是1时必填 |
| card_action.appid | String | 否 | 跳转事件的小程序的appid，card_action.type是2时必填 |
| card_action.pagepath | String | 否 | 跳转事件的小程序的pagepath，card_action.type是2时选填 |

### 图文展示模版卡片
```json
{
    "msgtype":"template_card",
    "template_card":{
        "card_type":"news_notice",
        "source":{
            "icon_url":"https://wework.qpic.cn/wwpic/252813_jOfDHtcISzuodLa_1629280209/0",
            "desc":"企业微信",
            "desc_color":0
        },
        "main_title":{
            "title":"欢迎使用企业微信",
            "desc":"您的好友正在邀请您加入企业微信"
        },
        "card_image":{
            "url":"https://wework.qpic.cn/wwpic/354393_4zpkKXd7SrGMvfg_1629280616/0",
            "aspect_ratio":2.25
        },
        "image_text_area":{
            "type":1,
            "url":"https://work.weixin.qq.com",
            "title":"欢迎使用企业微信",
            "desc":"您的好友正在邀请您加入企业微信",
            "image_url":"https://wework.qpic.cn/wwpic/354393_4zpkKXd7SrGMvfg_1629280616/0"
        },
        "quote_area":{
            "type":1,
            "url":"https://work.weixin.qq.com/?from=openApi",
            "appid":"APPID",
            "pagepath":"PAGEPATH",
            "title":"引用文本标题",
            "quote_text":"Jack：企业微信真的很好用~\nBalian：超级好的一款软件！"
        },
        "vertical_content_list":[\
            {
                "title":"惊喜红包等你来拿",
                "desc":"下载企业微信还能抢红包！"
            }
        ],
        "horizontal_content_list":[\
            {
                "keyname":"邀请人",
                "value":"张三"
            },
            {
                "keyname":"企微官网",
                "value":"点击访问",
                "type":1,
                "url":"https://work.weixin.qq.com/?from=openApi"
            },
            {
                "keyname":"企微下载",
                "value":"企业微信.apk",
                "type":2,
                "media_id":"MEDIAID"
            }
        ],
        "jump_list":[\
            {
                "type":1,
                "url":"https://work.weixin.qq.com/?from=openApi",
                "title":"企业微信官网"
            },
            {
                "type":2,
                "appid":"APPID",
                "pagepath":"PAGEPATH",
                "title":"跳转小程序"
            }
        ],
        "card_action":{
            "type":1,
            "url":"https://work.weixin.qq.com/?from=openApi",
            "appid":"APPID",
            "pagepath":"PAGEPATH"
        }
    }
}
```

请求参数

| 参数 | 类型 | 必须 | 说明 |
|---|---|---|---|
| msgtype | String | 是 | 模版卡片的消息类型为template_card |
| template_card | Object | 是 | 具体的模版卡片参数 |

template_card的参数说明

| 参数 | 类型 | 必须 | 说明 |
|---|---|---|---|
| card_type | String | 是 | 模版卡片的模版类型，图文展示模版卡片的类型为news_notice |
| source | Object | 否 | 卡片来源样式信息，不需要来源样式可不填写 |
| source.icon_url | String | 否 | 来源图片的url |
| source.desc | String | 否 | 来源图片的描述，建议不超过13个字 |
| source.desc_color | Int | 否 | 来源文字的颜色，目前支持：0(默认) 灰色，1 黑色，2 红色，3 绿色 |
| main_title | Object | 是 | 模版卡片的主要内容，包括一级标题和标题辅助信息 |
| main_title.title | String | 是 | 一级标题，建议不超过26个字 |
| main_title.desc | String | 否 | 标题辅助信息，建议不超过30个字 |
| card_image | Object | 是 | 图片样式 |
| card_image.url | String | 是 | 图片的url |
| card_image.aspect_ratio | Float | 否 | 图片的宽高比，宽高比要小于2.25，大于1.3，不填该参数默认1.3 |
| image_text_area | Object | 否 | 左图右文样式 |
| image_text_area.type | Int | 否 | 左图右文样式区域点击事件，0或不填代表没有点击事件，1 代表跳转url，2 代表跳转小程序 |
| image_text_area.url | String | 否 | 点击跳转的url，image_text_area.type是1时必填 |
| image_text_area.appid | String | 否 | 点击跳转的小程序的appid，必须是与当前应用关联的小程序，image_text_area.type是2时必填 |
| image_text_area.pagepath | String | 否 | 点击跳转的小程序的pagepath，image_text_area.type是2时选填 |
| image_text_area.title | String | 否 | 左图右文样式的标题 |
| image_text_area.desc | String | 否 | 左图右文样式的描述 |
| image_text_area.image_url | String | 是 | 左图右文样式的图片url |
| quote_area | Object | 否 | 引用文献样式，建议不与关键数据共用 |
| quote_area.type | Int | 否 | 引用文献样式区域点击事件，0或不填代表没有点击事件，1 代表跳转url，2 代表跳转小程序 |
| quote_area.url | String | 否 | 点击跳转的url，quote_area.type是1时必填 |
| quote_area.appid | String | 否 | 点击跳转的小程序的appid，quote_area.type是2时必填 |
| quote_area.pagepath | String | 否 | 点击跳转的小程序的pagepath，quote_area.type是2时选填 |
| quote_area.title | String | 否 | 引用文献样式的标题 |
| quote_area.quote_text | String | 否 | 引用文献样式的引用文案 |
| vertical_content_list | Object[] | 否 | 卡片二级垂直内容，该字段可为空数组，但有数据的话需确认对应字段是否必填，列表长度不超过4 |
| vertical_content_list.title | String | 是 | 卡片二级标题，建议不超过26个字 |
| vertical_content_list.desc | String | 否 | 二级普通文本，建议不超过112个字 |
| horizontal_content_list | Object[] | 否 | 二级标题+文本列表，该字段可为空数组，但有数据的话需确认对应字段是否必填，列表长度不超过6 |
| horizontal_content_list.type | Int | 否 | 模版卡片的二级标题信息内容支持的类型，1是url，2是文件附件，3 代表点击跳转成员详情 |
| horizontal_content_list.keyname | String | 是 | 二级标题，建议不超过5个字 |
| horizontal_content_list.value | String | 否 | 二级文本，如果horizontal_content_list.type是2，该字段代表文件名称（要包含文件类型），建议不超过26个字 |
| horizontal_content_list.url | String | 否 | 链接跳转的url，horizontal_content_list.type是1时必填 |
| horizontal_content_list.media_id | String | 否 | 附件的media_id，horizontal_content_list.type是2时必填 |
| horizontal_content_list.userid | String | 否 | 成员详情的userid，horizontal_content_list.type是3时必填 |
| jump_list | Object[] | 否 | 跳转指引样式的列表，该字段可为空数组，但有数据的话需确认对应字段是否必填，列表长度不超过3 |
| jump_list.type | Int | 否 | 跳转链接类型，0或不填代表不是链接，1 代表跳转url，2 代表跳转小程序 |
| jump_list.title | String | 是 | 跳转链接样式的文案内容，建议不超过13个字 |
| jump_list.url | String | 否 | 跳转链接的url，jump_list.type是1时必填 |
| jump_list.appid | String | 否 | 跳转链接的小程序的appid，jump_list.type是2时必填 |
| jump_list.pagepath | String | 否 | 跳转链接的小程序的pagepath，jump_list.type是2时选填 |
| card_action | Object | 是 | 整体卡片的点击跳转事件，news_notice模版卡片中该字段为必填项 |
| card_action.type | Int | 是 | 卡片跳转类型，1 代表跳转url，2 代表打开小程序。news_notice模版卡片中该字段取值范围为[1,2] |
| card_action.url | String | 否 | 跳转事件的url，card_action.type是1时必填 |
| card_action.appid | String | 否 | 跳转事件的小程序的appid，card_action.type是2时必填 |
| card_action.pagepath | String | 否 | 跳转事件的小程序的pagepath，card_action.type是2时选填 |

## 消息发送频率限制

每个机器人发送的消息不能超过20条/分钟。

## 文件上传接口
素材上传得到media_id，该media_id仅三天内有效
media_id只能是对应上传文件的机器人可以使用

请求方式：POST（HTTPS）
请求地址：https://qyapi.weixin.qq.com/cgi-bin/webhook/upload_media?key=KEY&type=TYPE

使用multipart/form-data POST上传文件或语音， 文件标识名为"media"
参数说明：

| 参数 | 必须 | 说明 |
|---|---|---|
| key | 是 | 调用接口凭证, 机器人webhookurl中的key参数 |
| type | 是 | 文件类型，分别有语音(voice)和普通文件(file) |

POST的请求包中，form-data中媒体文件标识，应包含有 filename、filelength、content-type等信息

filename标识文件展示的名称。比如，使用该media_id发消息时，展示的文件名由该字段控制

请求示例：

```
POST https://qyapi.weixin.qq.com/cgi-bin/webhook/upload_media?key=693a91f6-7xxx-4bc4-97a0-0ec2sifa5aaa&type=file HTTP/1.1
Content-Type: multipart/form-data; boundary=-------------------------acebdf13572468
Content-Length: 220

---------------------------acebdf13572468
Content-Disposition: form-data; name="media";filename="wework.txt"; filelength=6
Content-Type: application/octet-stream

mytext
---------------------------acebdf13572468--
```


返回数据：

```json
{
   "errcode": 0,
   "errmsg": "ok",
   "type": "file",
   "media_id": "1G6nrLmr5EC3MMb_-zK1dDdzmd0p7cNliYu9V5w7o8K0",
   "created_at": "1380000000"
}
```


参数说明：

| 参数 | 说明 |
|---|---|
| type | 文件类型，分别有语音(voice)和普通文件(file) |
| media_id | 媒体文件上传后获取的唯一标识，3天内有效 |
| created_at | 媒体文件上传时间戳 |

上传的文件限制：
所有类型的文件大小均要求大于5个字节

普通文件(file)：文件大小不超过20M
语音(voice)：文件大小不超过2M，播放长度不超过60s，仅支持AMR格式

上一篇
发送「学校通知」
下一篇
接收与回复消息
快速入门
概述
简易教程
服务端API
开发指南
基础
账号ID
通讯录管理
身份验证
企业互联
上下游
安全管理
消息推送
概述
发送应用消息
更新模版卡片消息
撤回应用消息
接收消息与事件
概述
消息格式
事件格式
被动回复消息格式
应用发送消息到群聊会话
概述
创建群聊会话
修改群聊会话
获取群聊会话
应用推送消息
家校消息推送
发送「学校通知」
群机器人
群机器人配置说明
智能机器人
接收与回复消息
应用管理
素材管理
电子发票
数据与智能专区
连接微信
客户联系
微信客服
企业支付
小程序接入对外收款
会话内容存档
家校沟通
家校应用
政民沟通
办公
邮件
文档
日程
会议
微盘
直播
公费电话
打卡
审批
汇报
人事助手
会议室
高级功能
紧急通知应用
客户端API
小程序
JS-SDK
移动端SDK
群机器人
工具与资源
开发者工具
样式库 WeUI for Work
设计资源下载
加解密库下载与返回码
接口代码参考示例
移动端SDK资源下载
企业微信应用深色模式色值表
附录
加解密方案说明
访问频率限制
全局错误码
企业规模与行业信息
常见问题 - FAQ
与企业号接口差异
深色模式适配指南
更新日志
更新日志
联系我们
联系我们
