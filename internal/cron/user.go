package cron

import (
	"go-demo/config/di"
	"go-demo/pkg/dbcache"
	"go-demo/pkg/dbx"
)

// 用户相关计划任务 DEMO 这里定义一个空结构体用于为大量的cron方法做分类
type user struct{}

// User 这里仅需结构体零值
var User user

// DeleteUsers 批量删除用户
//
//	counts 为需要删除的数量.
func (user) DeleteUsers(counts int) {
	userIds, err := dbx.FetchColumn(di.DemoDb(), "SELECT user_id FROM t_users ORDER BY user_id LIMIT ?", counts)
	if err != nil {
		return
	}
	for _, userId := range userIds {
		userId := userId
		di.WorkerPool().Submit(func() {
			_, _ = dbcache.Delete(di.CacheRedis(), di.DemoDb(), "t_users", "user_id = ?", userId)
		})
	}
}
