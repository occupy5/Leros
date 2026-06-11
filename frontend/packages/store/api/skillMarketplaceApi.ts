import { apiClient } from "./client";
import type { BackendDataResponse } from "./types";

export interface SkillMarketplaceItem {
  source_type: string;
  skill_id: string;
  name: string;
  description: string;
  version: string;
  author: string;
  category: string;
  tags: string[];
  icon: string;
  installs: number;
}

export interface SearchSkillMarketplaceResponse {
  items: SkillMarketplaceItem[];
  total: number;
  warnings?: Array<{ source_type: string; message: string }>;
}

export interface SearchSkillMarketplaceParams {
  keyword?: string;
  category?: string;
  source_types?: string[];
  offset?: number;
  limit?: number;
}

function cleanParams(
  params: SearchSkillMarketplaceParams,
): Record<string, string | number | boolean> {
  const result: Record<string, string | number | boolean> = {};
  if (params.keyword) result.keyword = params.keyword;
  if (params.category) result.category = params.category;
  if (params.offset !== undefined) result.offset = params.offset;
  if (params.limit !== undefined) result.limit = params.limit;
  return result;
}

export const skillMarketplaceApi = {
  search: (params: SearchSkillMarketplaceParams) =>
    apiClient.get<BackendDataResponse<SearchSkillMarketplaceResponse>>(
      "/skill-marketplace/search",
      { params: cleanParams(params) },
    ),
};
