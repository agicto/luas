'use client';

import {
  useMutation,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query';

import { LOCAL_ERROR_HANDLING_META } from '@/config/query-meta';
import { organizationService } from '@/features/organization/services/organization-service';
import type {
  AcceptOrganizationInvitationInput,
  CreateOrganizationInvitationInput,
  CreateOrganizationInput,
  Organization,
  OrganizationContext,
  OrganizationInvitationPage,
  OrganizationMember,
  OrganizationMemberPage,
  OrganizationPage,
  TransferOrganizationOwnershipInput,
  UpdateOrganizationMemberInput,
  UpdateOrganizationInput,
} from '@/features/organization/types';

export const organizationKeys = {
  all: ['organizations'] as const,
  list: () => [...organizationKeys.all, 'list'] as const,
  detail: (organizationId: number) =>
    [...organizationKeys.all, 'detail', organizationId] as const,
  context: (organizationId: number) =>
    [...organizationKeys.all, 'context', organizationId] as const,
  members: (organizationId: number) =>
    [...organizationKeys.all, 'members', organizationId] as const,
  invitations: (organizationId: number) =>
    [...organizationKeys.all, 'invitations', organizationId] as const,
};

export function useOrganizations(enabled = true) {
  return useQuery({
    queryKey: organizationKeys.list(),
    queryFn: () => organizationService.list(),
    enabled,
    staleTime: 30_000,
  });
}

export function useOrganization(organizationId: number) {
  return useQuery({
    queryKey: organizationKeys.detail(organizationId),
    queryFn: () => organizationService.get(organizationId),
    enabled: Number.isSafeInteger(organizationId) && organizationId > 0,
    staleTime: 30_000,
  });
}

export function useOrganizationContext(organizationId: number) {
  return useQuery({
    queryKey: organizationKeys.context(organizationId),
    queryFn: () => organizationService.resolveContext(organizationId),
    enabled: Number.isSafeInteger(organizationId) && organizationId > 0,
    staleTime: 15_000,
  });
}

export function useOrganizationMembers(organizationId: number) {
  return useQuery({
    queryKey: organizationKeys.members(organizationId),
    queryFn: () => organizationService.listMembers(organizationId),
    enabled: validId(organizationId),
    staleTime: 15_000,
  });
}

export function useOrganizationInvitations(
  organizationId: number,
  enabled = true
) {
  return useQuery({
    queryKey: organizationKeys.invitations(organizationId),
    queryFn: () => organizationService.listInvitations(organizationId),
    enabled: enabled && validId(organizationId),
    staleTime: 15_000,
  });
}

export function useCreateOrganization() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: CreateOrganizationInput) =>
      organizationService.create(input),
    meta: LOCAL_ERROR_HANDLING_META,
    onSuccess: (organization) => {
      queryClient.setQueryData(
        organizationKeys.detail(organization.id),
        organization
      );
      queryClient.invalidateQueries({ queryKey: organizationKeys.list() });
    },
  });
}

export function useUpdateOrganization(organizationId: number) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: UpdateOrganizationInput) =>
      organizationService.update(organizationId, input),
    meta: LOCAL_ERROR_HANDLING_META,
    onMutate: async (input) => {
      await Promise.all([
        queryClient.cancelQueries({ queryKey: organizationKeys.list() }),
        queryClient.cancelQueries({ queryKey: organizationKeys.detail(organizationId) }),
        queryClient.cancelQueries({ queryKey: organizationKeys.context(organizationId) }),
      ]);
      const previousList = queryClient.getQueryData<OrganizationPage>(
        organizationKeys.list()
      );
      const previousDetail = queryClient.getQueryData<Organization>(
        organizationKeys.detail(organizationId)
      );
      const previousContext = queryClient.getQueryData<OrganizationContext>(
        organizationKeys.context(organizationId)
      );

      queryClient.setQueryData<OrganizationPage>(organizationKeys.list(), (current) =>
        current
          ? {
              ...current,
              items: current.items.map((organization) =>
                organization.id === organizationId
                  ? { ...organization, name: input.name }
                  : organization
              ),
            }
          : current
      );
      queryClient.setQueryData<Organization>(
        organizationKeys.detail(organizationId),
        (current) => current ? { ...current, name: input.name } : current
      );
      queryClient.setQueryData<OrganizationContext>(
        organizationKeys.context(organizationId),
        (current) => current
          ? { ...current, organization_name: input.name }
          : current
      );

      return { previousList, previousDetail, previousContext };
    },
    onError: (_error, _input, context) => {
      if (!context) return;
      queryClient.setQueryData(organizationKeys.list(), context.previousList);
      queryClient.setQueryData(
        organizationKeys.detail(organizationId),
        context.previousDetail
      );
      queryClient.setQueryData(
        organizationKeys.context(organizationId),
        context.previousContext
      );
    },
    onSuccess: (organization) => {
      queryClient.setQueryData(organizationKeys.detail(organizationId), organization);
      queryClient.setQueryData<OrganizationContext>(
        organizationKeys.context(organizationId),
        (current) => current
          ? { ...current, organization_name: organization.name }
          : current
      );
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: organizationKeys.list() });
      queryClient.invalidateQueries({ queryKey: organizationKeys.detail(organizationId) });
      queryClient.invalidateQueries({ queryKey: organizationKeys.context(organizationId) });
    },
  });
}

export function useUpdateOrganizationMember(organizationId: number) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      memberId,
      input,
    }: {
      memberId: number;
      input: UpdateOrganizationMemberInput;
    }) => organizationService.changeMemberRole(organizationId, memberId, input),
    meta: LOCAL_ERROR_HANDLING_META,
    onSuccess: (member) => {
      updateMemberPage(queryClient, organizationId, member);
    },
    onSettled: () => {
      queryClient.invalidateQueries({
        queryKey: organizationKeys.members(organizationId),
      });
    },
  });
}

export function useRemoveOrganizationMember(organizationId: number) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (memberId: number) =>
      organizationService.removeMember(organizationId, memberId),
    meta: LOCAL_ERROR_HANDLING_META,
    onSuccess: (_value, memberId) => {
      queryClient.setQueryData<OrganizationMemberPage>(
        organizationKeys.members(organizationId),
        (current) =>
          current
            ? {
                ...current,
                items: current.items.filter((member) => member.id !== memberId),
                meta: {
                  ...current.meta,
                  total: Math.max(0, current.meta.total - 1),
                },
              }
            : current
      );
    },
    onSettled: () => {
      queryClient.invalidateQueries({
        queryKey: organizationKeys.members(organizationId),
      });
    },
  });
}

export function useTransferOrganizationOwnership(organizationId: number) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: TransferOrganizationOwnershipInput) =>
      organizationService.transferOwnership(organizationId, input),
    meta: LOCAL_ERROR_HANDLING_META,
    onSuccess: (transfer) => {
      updateMemberPage(queryClient, organizationId, transfer.previous_owner);
      updateMemberPage(queryClient, organizationId, transfer.new_owner);
      setOrganizationRole(queryClient, organizationId, 'admin');
    },
    onSettled: () => {
      queryClient.invalidateQueries({
        queryKey: organizationKeys.members(organizationId),
      });
      queryClient.invalidateQueries({ queryKey: organizationKeys.list() });
      queryClient.invalidateQueries({
        queryKey: organizationKeys.context(organizationId),
      });
    },
  });
}

export function useCreateOrganizationInvitation(organizationId: number) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: CreateOrganizationInvitationInput) =>
      organizationService.invite(organizationId, input),
    meta: LOCAL_ERROR_HANDLING_META,
    onSuccess: ({ invitation }) => {
      queryClient.setQueryData<OrganizationInvitationPage>(
        organizationKeys.invitations(organizationId),
        (current) =>
          current
            ? {
                ...current,
                items: [
                  invitation,
                  ...current.items.filter((item) => item.id !== invitation.id),
                ],
                meta: { ...current.meta, total: current.meta.total + 1 },
              }
            : current
      );
    },
    onSettled: () => {
      queryClient.invalidateQueries({
        queryKey: organizationKeys.invitations(organizationId),
      });
    },
  });
}

export function useRevokeOrganizationInvitation(organizationId: number) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (invitationId: number) =>
      organizationService.revokeInvitation(organizationId, invitationId),
    meta: LOCAL_ERROR_HANDLING_META,
    onSettled: () => {
      queryClient.invalidateQueries({
        queryKey: organizationKeys.invitations(organizationId),
      });
    },
  });
}

export function useAcceptOrganizationInvitation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: AcceptOrganizationInvitationInput) =>
      organizationService.acceptInvitation(input),
    meta: LOCAL_ERROR_HANDLING_META,
    onSuccess: (organization) => {
      queryClient.setQueryData(
        organizationKeys.detail(organization.id),
        organization
      );
      queryClient.invalidateQueries({ queryKey: organizationKeys.list() });
    },
  });
}

function validId(value: number): boolean {
  return Number.isSafeInteger(value) && value > 0;
}

function updateMemberPage(
  queryClient: ReturnType<typeof useQueryClient>,
  organizationId: number,
  member: OrganizationMember
): void {
  queryClient.setQueryData<OrganizationMemberPage>(
    organizationKeys.members(organizationId),
    (current) =>
      current
        ? {
            ...current,
            items: current.items.map((item) =>
              item.id === member.id ? member : item
            ),
          }
        : current
  );
}

function setOrganizationRole(
  queryClient: ReturnType<typeof useQueryClient>,
  organizationId: number,
  role: Organization['role']
): void {
  queryClient.setQueryData<OrganizationPage>(
    organizationKeys.list(),
    (current) =>
      current
        ? {
            ...current,
            items: current.items.map((organization) =>
              organization.id === organizationId
                ? { ...organization, role }
                : organization
            ),
          }
        : current
  );
  queryClient.setQueryData<Organization>(
    organizationKeys.detail(organizationId),
    (current) => (current ? { ...current, role } : current)
  );
  queryClient.setQueryData<OrganizationContext>(
    organizationKeys.context(organizationId),
    (current) => (current ? { ...current, role } : current)
  );
}
