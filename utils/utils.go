package utils

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gohouse/gorose/v2"
	"math"
	"math/rand"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Md5 md5加密字符串
func Md5(str string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(str)))
}

// GenToken 生成一个token字符串
func GenToken() string {
	seed := strconv.FormatInt(time.Now().UnixNano(), 10) + strconv.Itoa(rand.Int())

	return Md5(seed)
}

// GetJsonBody 获取Json参数
// @param patterns ["paramKey:paramName:paramType:paramPattern"] paramPattern +必填不可为空, *选填可以为空, ?选填不可为空
// 参数异常时方法会向客户端返回4xx错误, 调用方法时捕获到error直接结束业务逻辑即可
func GetJsonBody(c *gin.Context, patterns []string) (res map[string]interface{}, resErr error) {
	jsonBody := make(map[string]interface{})
	_ = c.ShouldBindJSON(&jsonBody) // 这里的error不要处理, 因为空body会报error
	res = make(map[string]interface{})

	for _, pattern := range patterns {
		patternAtoms := strings.Split(pattern, ":")
		required := true
		allowEmpty := false
		if "+" == patternAtoms[3] {
			required = true
			allowEmpty = false
		} else if "*" == patternAtoms[3] {
			required = false
			allowEmpty = true
		} else if "?" == patternAtoms[3] {
			required = false
			allowEmpty = false
		}

		paramValue, ok := jsonBody[patternAtoms[0]]
		if !ok {
			if required {
				c.JSON(400, gin.H{"status": "emptyParam", "message": fmt.Sprintf("%s不得为空", patternAtoms[1])})
				resErr = errors.New("emptyParam")
				return
			} else {
				continue
			}
		}

		res[patternAtoms[0]], resErr = FilterParam(c, patternAtoms[1], paramValue, patternAtoms[2], allowEmpty)
		if resErr != nil {
			return
		}
	}

	return
}

// GetQueries 获取Query参数
// @param patterns ["paramKey:paramName:paramType:defaultValue"] defaultValue为nil时参数必填
// 参数异常时方法会向客户端返回4xx错误, 调用方法时捕获到error直接结束业务逻辑即可
func GetQueries(c *gin.Context, patterns []string) (res map[string]interface{}, resErr error) {
	res = make(map[string]interface{})

	for _, pattern := range patterns {
		patternAtoms := strings.Split(pattern, ":")
		allowEmpty := false
		if `""` == patternAtoms[3] { // 默认值""表示空字符串
			patternAtoms[3] = ""
			allowEmpty = true
		}
		paramValue := c.Query(patternAtoms[0])
		if "" == paramValue {
			if "nil" == patternAtoms[3] { // 必填
				c.JSON(400, gin.H{"status": "emptyParam", "message": fmt.Sprintf("%s不得为空", patternAtoms[1])})
				resErr = errors.New("emptyParam")
				return
			} else {
				paramValue = patternAtoms[3]
			}
		}

		res[patternAtoms[0]], resErr = FilterParam(c, patternAtoms[1], paramValue, patternAtoms[2], allowEmpty)
		if resErr != nil {
			return
		}
	}

	return
}

// FilterParam 校验参数类型
// @param paramType int整型64位, +int正整型64位, !-int非负整型64位, string字符串, []枚举, array数组
// 参数异常时方法会向客户端返回4xx错误, 调用方法时捕获到error直接结束业务逻辑即可
func FilterParam(c *gin.Context, paramName string, paramValue interface{}, paramType string, allowEmpty bool) (resValue interface{}, resErr error) {
	valueType := reflect.TypeOf(paramValue).String()

	if "int" == paramType[len(paramType)-3:] { // 整型
		var intValue int64
		var err error
		var stringValue string // 先统一转字符串再转整型, 这样小数就不允许输入了
		if "string" == valueType {
			stringValue = strings.TrimSpace(paramValue.(string))
			if "" == stringValue && !allowEmpty {
				c.JSON(400, gin.H{"status": "emptyParam", "message": fmt.Sprintf("%s不得为空", paramName)})
				resErr = errors.New("emptyParam")
				return
			}
		} else if "float64" == valueType {
			stringValue = fmt.Sprintf("%v", paramValue)
		} else {
			c.JSON(400, gin.H{"status": "InvalidParam", "message": fmt.Sprintf("%s不正确", paramName)})
			resErr = errors.New("InvalidParam")
			return
		}
		intValue, err = strconv.ParseInt(stringValue, 10, 64) // 转整型64位
		if err != nil {
			c.JSON(400, gin.H{"status": "InvalidParam", "message": fmt.Sprintf("%s不正确", paramName)})
			resErr = errors.New("InvalidParam")
			return
		}
		if ("+int" == paramType && intValue <= 0) || ("!-int" == paramType && intValue < 0) { // 范围过滤
			c.JSON(400, gin.H{"status": "InvalidParam", "message": fmt.Sprintf("%s不正确", paramName)})
			resErr = errors.New("InvalidParam")
			return
		}
		resValue = intValue
		return

	} else if "string" == paramType { // 字符串, 去首尾空格
		if "string" == valueType {
			stringValue := strings.TrimSpace(paramValue.(string))
			if "" == stringValue && !allowEmpty {
				c.JSON(400, gin.H{"status": "emptyParam", "message": fmt.Sprintf("%s不得为空", paramName)})
				resErr = errors.New("emptyParam")
				return
			}
			resValue = stringValue
			return
		} else if "float64" == valueType {
			stringValue := fmt.Sprintf("%v", paramValue)
			resValue = stringValue
			return
		} else {
			c.JSON(400, gin.H{"status": "InvalidParam", "message": fmt.Sprintf("%s不正确", paramName)})
			resErr = errors.New("InvalidParam")
			return
		}

	} else if EnumMark := paramType[0:1]; "[" == EnumMark { // 枚举, 支持数字与字符串混合枚举
		var enum []interface{}
		if err := json.Unmarshal([]byte(paramType), &enum); err != nil {
			panic(err)
		}
		for _, value := range enum {
			enumType := reflect.TypeOf(enum[0]).String()
			if enumType == valueType && paramValue == value {
				resValue = value
				return
			}
			if "float64" == valueType {
				stringValue := fmt.Sprintf("%v", paramValue)
				if stringValue == value {
					resValue = value
					return
				}
			} else if "string" == valueType {
				floatValue, err := strconv.ParseFloat(paramValue.(string), 64)
				if err != nil {
					panic(err)
				}
				resValue = floatValue
				return
			} else {
				c.JSON(400, gin.H{"status": "InvalidParam", "message": fmt.Sprintf("%s不正确", paramName)})
				resErr = errors.New("InvalidParam")
				return
			}
		}
		c.JSON(400, gin.H{"status": "InvalidParam", "message": fmt.Sprintf("%s不正确", paramName)})
		resErr = errors.New("InvalidParam")
		return

	} else if "array" == paramType { // 数组
		if "[]interface {}" == valueType {
			resValue = paramValue
			return
		} else {
			c.JSON(400, gin.H{"status": "InvalidParam", "message": fmt.Sprintf("%s不正确", paramName)})
			resErr = errors.New("InvalidParam")
			return
		}
	}

	c.JSON(400, gin.H{"status": "UndefinedParamType", "message": fmt.Sprintf("未知数据类型: %s", paramName)})
	resErr = errors.New("UndefinedParamType")
	return
}

// GetPageItems 获取分页数据
// @param query {"ginContext": *gin.Context, "db": gorose.IOrm, "select": string, "from": string, "where": string, "groupBy" => string, "having" => string, "orderBy": string}
// @return {"page": int64, "per_page": int64, "total_page": int64, "total_counts": int64, "items": []map[string]interface{}}
// 出现异常时方法会向客户端返回4xx错误, 调用方法捕获到error直接结束业务逻辑即可
func GetPageItems(query map[string]interface{}) (res map[string]interface{}, resErr error) {
	queries, resErr := GetQueries(query["ginContext"].(*gin.Context), []string{"page:页码:+int:1", "per_page:页大小:+int:12"})
	if resErr != nil {
		return
	}

	res = make(map[string]interface{})

	bindParams, ok := query["bindParams"].([]interface{}) // 参数绑定
	if !ok {
		bindParams = []interface{}{}
	}

	where := query["where"].(string)

	var countSql string
	groupBy, ok := query["groupBy"].(string) // GROUP BY存在总记录数计算方式会不同
	if ok {
		where += " " + groupBy
		having, ok := query["having"].(string)
		if ok {
			where += " " + having
		}
		countSql = fmt.Sprintf("SELECT COUNT(*) AS counts FROM (SELECT %s FROM %s WHERE %s) AS t", query["select"], query["from"], where)
	} else {
		countSql = fmt.Sprintf("SELECT COUNT(*) AS counts FROM %s WHERE %s", query["from"], where)
	}
	counts, err := query["db"].(gorose.IOrm).Query(countSql, bindParams...) // 计算总记录数
	if err != nil {
		panic(err)
	}
	if 0 == counts[0]["counts"].(int64) { // 没有数据
		res = map[string]interface{}{
			"page":         queries["page"],
			"per_page":     queries["per_page"],
			"total_pages":  0,
			"total_counts": 0,
			"items":        []gorose.Data{},
		}
		return
	}

	sql := fmt.Sprintf("SELECT %s FROM %s WHERE %s", query["select"], query["from"], where)
	orderBy, ok := query["orderBy"]
	if ok {
		sql += fmt.Sprintf(" ORDER BY %s", orderBy)
	}
	offset := (queries["page"].(int64) - 1) * queries["per_page"].(int64)
	sql += fmt.Sprintf(" LIMIT %d, %d", offset, queries["per_page"])
	items, err := query["db"].(gorose.IOrm).Query(sql, bindParams...)
	if err != nil {
		panic(err)
	}
	res = map[string]interface{}{
		"page":         queries["page"],
		"per_page":     queries["per_page"],
		"total_pages":  math.Ceil(float64(counts[0]["counts"].(int64)) / float64(queries["per_page"].(int64))),
		"total_counts": counts[0]["counts"],
		"items":        items,
	}
	return
}
