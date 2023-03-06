package relation

import (
	"OpenIM/pkg/common/db/table/relation"
	"OpenIM/pkg/common/tracelog"
	"OpenIM/pkg/utils"
	"context"
	"gorm.io/gorm"
)

func NewObjectHash(db *gorm.DB) relation.ObjectHashModelInterface {
	return &ObjectHashGorm{
		DB: db,
	}
}

type ObjectHashGorm struct {
	DB *gorm.DB
}

func (o *ObjectHashGorm) NewTx(tx any) relation.ObjectHashModelInterface {
	return &ObjectHashGorm{
		DB: tx.(*gorm.DB),
	}
}

func (o *ObjectHashGorm) Take(ctx context.Context, hash string, engine string) (oh *relation.ObjectHashModel, err error) {
	defer func() {
		tracelog.SetCtxDebug(ctx, utils.GetFuncName(1), err, "hash", hash, "engine", engine, "objectHash", oh)
	}()
	oh = &relation.ObjectHashModel{}
	return oh, utils.Wrap1(o.DB.Where("hash = ? and engine = ?", hash, engine).Take(oh).Error)
}

func (o *ObjectHashGorm) Create(ctx context.Context, h []*relation.ObjectHashModel) (err error) {
	defer func() {
		tracelog.SetCtxDebug(ctx, utils.GetFuncName(1), err, "objectHash", h)
	}()
	return utils.Wrap1(o.DB.Create(h).Error)
}