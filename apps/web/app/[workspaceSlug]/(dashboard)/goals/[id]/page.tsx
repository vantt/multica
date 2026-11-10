"use client";

import { use } from "react";
import { GoalsPage } from "@multica/views/goals";

export default function Page({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  return <GoalsPage goalId={id} />;
}
