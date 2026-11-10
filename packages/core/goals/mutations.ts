import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../api";
import { useWorkspaceId } from "../hooks";
import { goalKeys } from "./queries";
import type { GoalWriteRequest } from "./types";

function useInvalidateGoals() {
  const qc = useQueryClient();
  const wsId = useWorkspaceId();
  return () => qc.invalidateQueries({ queryKey: goalKeys.all(wsId) });
}

export function useCreateGoal() {
  const invalidate = useInvalidateGoals();
  return useMutation({ mutationFn: (body: GoalWriteRequest) => api.createGoal(body), onSuccess: invalidate });
}

export function useUpdateGoal() {
  const invalidate = useInvalidateGoals();
  return useMutation({
    mutationFn: ({ id, ...body }: GoalWriteRequest & { id: string }) => api.updateGoal(id, body),
    onSuccess: invalidate,
  });
}

export function useAddGoalValue() {
  const invalidate = useInvalidateGoals();
  return useMutation({
    mutationFn: ({ id, value, note }: { id: string; value: number; note?: string }) => api.addGoalValue(id, value, note ?? ""),
    onSuccess: invalidate,
  });
}
