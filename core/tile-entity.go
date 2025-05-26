package core

type TileEntity interface {
	Draw(realX, realY int32)
	Update()
}

