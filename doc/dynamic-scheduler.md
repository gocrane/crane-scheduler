# Dynamic-scheduler

Native strategies of kube-scheduler can only schedule pods by resource allocation(pod requests), which can easily cause a series of load uneven problems.

To solve this problem, Dynamic scheduler builds a simple but efficient model based on actual node utilization data.

In brief, Dynamic will filter high load nodes at scheduling Filter stage, and prioritize candidates by real resource usages, instead of resource requests at Score stage.
