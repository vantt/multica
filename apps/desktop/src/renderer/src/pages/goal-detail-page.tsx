import { useParams } from "react-router-dom";
import { GoalsPage } from "@multica/views/goals";
import { useDocumentTitle } from "@/hooks/use-document-title";

export function GoalDetailPage() {
  const { id } = useParams<{ id: string }>();
  useDocumentTitle("Goal");
  if (!id) return null;
  return <GoalsPage goalId={id} />;
}
