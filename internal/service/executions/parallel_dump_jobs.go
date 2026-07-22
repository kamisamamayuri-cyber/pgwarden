package executions

func (s *Service) ResolveParallelDumpJobs(configured int16) int {
	jobs := int(configured)
	if jobs == 0 {
		jobs = s.env.PBW_DUMP_PARALLEL_JOBS
	}
	return max(2, min(jobs, 16))
}
