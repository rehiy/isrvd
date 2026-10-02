package cstore

import (
	"bytes"
	"context"
	"time"

	"github.com/rehiy/libgo/logman"
)

// Watch 把事件作为变更提示，读取并发送最新状态；首次建立/重连时补偿连接间隙。
// 读取失败会重试，连续相同状态不重复发送；不重放所有历史事件。
func (e *EtcdStore) Watch(ctx context.Context, key string) <-chan Event {
	out := make(chan Event, 8)
	watchEvents, watchErrs := e.client.Watch(ctx, e.etcdKey(key))
	go func() {
		defer close(out)
		var last []byte
		known := false
		retry := time.NewTimer(time.Hour)
		retry.Stop()
		defer retry.Stop()
		var retryCh <-chan time.Time
		for {
			select {
			case <-ctx.Done():
				return
			case _, ok := <-watchEvents:
				if !ok {
					return
				}
			case <-retryCh:
			case err, ok := <-watchErrs:
				if !ok {
					watchErrs = nil
				} else {
					logman.Warn("cstore: Watch 连接错误", "key", key, "error", err)
				}
				continue
			}
			value, err := e.read(ctx, key)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				logman.Warn("cstore: Watch 同步失败，稍后重试", "key", key, "error", err)
				retry.Reset(time.Second)
				retryCh = retry.C
				continue
			}
			retry.Stop()
			retryCh = nil
			if known && (value == nil) == (last == nil) && bytes.Equal(value, last) {
				continue
			}
			event := Event{Key: key, Type: EventPut, Value: value}
			if value == nil {
				event.Type = EventDelete
			}
			select {
			case out <- event:
				last, known = value, true
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}
