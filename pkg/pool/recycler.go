package pool

func (p *Pool) Recycle(vm *PooledVM) {
	p.destroy(vm)
}
