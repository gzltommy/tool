package uuid

import "github.com/bwmarrin/snowflake"

var snowflakeNode *snowflake.Node

func GenerateUUID() int64 {
	return int64(snowflakeNode.Generate())
}

func InitNode(node *snowflake.Node) {
	snowflakeNode = node
}
