package tiles

type TileEntity interface {
	Draw(realX, realY int32)
	Update()
}

