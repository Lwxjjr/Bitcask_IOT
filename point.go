package core

// Row 包含一个数据点以及用于标识一种指标的属性。
type Row struct {
	// 指标的唯一名称。
	// 必须设置此字段。
	Metric string
	// 用于进一步详细标识的可选键值属性。
	Labels []Label
	// 必须设置此字段。
	DataPoint
}

// DataPoint 表示一个数据点，是时序数据的最小单位。
type DataPoint struct {
	// 实际值。必须设置此字段。
	Value float64
	// Unix 时间戳。
	Timestamp int64
}

const (
	// 标签名称的最大长度。
	//
	// 更长的名称会被截断。
	maxLabelNameLen = 256

	// 标签值的最大长度。
	//
	// 更长的值会被截断。
	maxLabelValueLen = 16 * 1024
)

// Label 是一个时序标签。
// 缺少名称或值的标签是无效的。
type Label struct {
	Name  string
	Value string
}

// marshalMetricName 通过编码标签来构建唯一的字节。
func marshalMetricName(metric string, labels []Label) string {
	// TODO
	return ""
}
