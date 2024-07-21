package main

import (
	"backend/lib"
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Config struct {
	lib.DatabaseConfig
	Port string
}

type Graph struct {
	Id   uint32 `gorm:"primary_key" json:"id"`
	Name string `json:"name"`
}

type Node struct {
	Id      uint32 `gorm:"primary_key" json:"id"`
	GraphId uint32 `json:"graph_id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Src     string `json:"src"`
}

type Link struct {
	Id      uint32 `gorm:"primary_key" json:"id"`
	GraphId uint32 `json:"graph_id"`
	Source  uint32 `json:"source"`
	Target  uint32 `json:"target"`
	Type    string `json:"type"`
}

type DuplicateNode struct {
	Id    uint32 `gorm:"primary_key" json:"id"`
	Node1 uint32 `json:"node_1"`
	Node2 uint32 `json:"node_2"`
}

func main() {
	config := lib.LoadConfig[Config]()
	db := lib.NewDB(&config.DatabaseConfig, func(db *gorm.DB) error {
		return db.AutoMigrate(&Graph{}, &Node{}, &Link{}, &DuplicateNode{})
	})

	r := gin.Default()
	lib.APIBuilder(func(rg *gin.RouterGroup) *gin.RouterGroup {
		rg.GET("", lib.GetAll[Graph](db, nil))
		rg.GET("/by_graph/:gid", lib.GetAll[Graph](db, func(d *gorm.DB, ctx *gin.Context) *gorm.DB {
			return d.Where("graph_id = ?", ctx.Param("gid"))
		}))
		rg.POST("", lib.Create[Graph](db, nil))
		rg.POST("/new_kg", func(ctx *gin.Context) {
			req := new(struct {
				Graph Graph
				Nodes []Node
				Links []Link
			})
			if err := ctx.ShouldBindJSON(req); err != nil {
				log.Println("[new_kg] parse data err: ", err)
				ctx.AbortWithStatus(500)
			} else {
				// Get the current maximum Id values from the database
				var maxNodeId, maxLinkId uint32
				db.Model(&Node{}).Select("MAX(id)").Scan(&maxNodeId)
				db.Model(&Link{}).Select("MAX(id)").Scan(&maxLinkId)

				// Increment Id values for new entries
				newNodeId := maxNodeId + 1
				nodeIdMap := make(map[uint32]uint32) // Map to track old to new Id mapping

				for i := range req.Nodes {
					req.Nodes[i].Id = newNodeId
					nodeIdMap[uint32(i+1)] = newNodeId // Assuming original Ids start from 1
					newNodeId++
				}

				newLinkId := maxLinkId + 1
				for i := range req.Links {
					req.Links[i].Id = newLinkId
					req.Links[i].Source = nodeIdMap[req.Links[i].Source]
					req.Links[i].Target = nodeIdMap[req.Links[i].Target]
					newLinkId++
				}

				db.Create(&req.Graph)
				for i := range req.Nodes {
					req.Nodes[i].GraphId = req.Graph.Id
					db.Create(&req.Nodes[i])
				}
				for i := range req.Links {
					req.Links[i].GraphId = req.Graph.Id
					db.Create(&req.Links[i])
				}
				ctx.JSON(200, req)
			}
		})
		rg.PUT("/:id", lib.Update[Graph](db))
		rg.DELETE("/:id", lib.Delete[Graph](db))
		return rg
	})(r, "/graph")
	lib.AddCRUDNew[Node](r, "/node", db, nil, func(d *gorm.DB, ctx *gin.Context) *gorm.DB {
		return d.Where("graph_id = ?", ctx.Param("gid"))
	}, nil)
	lib.APIBuilder(func(rg *gin.RouterGroup) *gin.RouterGroup {
		rg.GET("", lib.GetAll[Link](db, nil))
		rg.GET("/by_graph/:gid", lib.GetAll[Link](db, func(d *gorm.DB, ctx *gin.Context) *gorm.DB {
			return d.Where("graph_id = ?", ctx.Param("gid"))
		}))
		rg.POST("", lib.Create[Link](db, nil))
		rg.PUT("/:id", lib.Update[Link](db))
		rg.DELETE("/:id", lib.Delete[Link](db))
		return rg
	})(r, "/link")
	lib.APIBuilder(func(rg *gin.RouterGroup) *gin.RouterGroup {
		rg.GET("", lib.GetAll[Link](db, nil))
		rg.GET("/:gid_a/:gid_b", lib.GetAll[Link](db, func(d *gorm.DB, ctx *gin.Context) *gorm.DB {
			return d.Where("graph_id_1 = ? AND graph_id_2 = ?", ctx.Param("gid_a"), ctx.Param("gid_b"))
		}))
		rg.POST("", lib.Create[Link](db, nil))
		rg.PUT("/:id", lib.Update[Link](db))
		rg.DELETE("/:id", lib.Delete[Link](db))
		return rg
	})(r, "/duplicated")

	r.Run(config.Port)
}
