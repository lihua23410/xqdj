package unit

import "xqdj/internal/replay"

// Presentation 是战斗机/场地包向开场编排返回的演出描述。
// 通过 unit 协议导出，生产者无需依赖 sim 或视频导出包。
type Presentation = replay.Presentation
