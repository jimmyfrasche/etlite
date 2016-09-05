package virt

import "strconv"

func (c *Machine) tmpSavepointName() string {
	s := c.saveprefix + strconv.Itoa(c.tmpSP)
	c.tmpSP++
	return s
}

func MkSavepointed(is []Instruction) Instruction {
	return func(c *Machine) error {
		sp := c.tmpSavepointName()
		if err := c.exec("SAVEPOINT " + sp); err != nil {
			return err
		}
		if err := c.Run(is); err != nil {
			if err := c.exec("ROLLBACK TO SAVEPOINT " + sp); err != nil {
				return err
			}
			return err
		}
		return c.exec("RELEASE " + sp)
	}
}
