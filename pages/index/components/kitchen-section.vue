<template>
	<view class="kitchen-section">
		<view class="kitchen-hero">
			<view class="kitchen-card" :class="{ 'kitchen-card--disabled': !currentKitchenName }" @tap="$emit('open-kitchen-selector')">
				<view class="kitchen-card__glow kitchen-card__glow--main"></view>
				<view class="kitchen-card__glow kitchen-card__glow--sage"></view>
				<view class="kitchen-card__header">
					<view class="kitchen-card__badge">
						<up-icon name="grid-fill" size="12" color="#5b4a3b"></up-icon>
						<text class="kitchen-card__badge-text">当前空间</text>
					</view>
					<view class="kitchen-card__switch">
						<text class="kitchen-card__switch-text">{{ canSwitchKitchen ? '切换' : kitchenConnectionLabel }}</text>
						<up-icon v-if="canSwitchKitchen" name="arrow-right" size="14" color="#7f7265"></up-icon>
						<view
							v-else
							class="kitchen-card__status-dot"
							:class="{ 'kitchen-card__status-dot--connected': isKitchenConnected }"
						></view>
					</view>
				</view>
				<view class="kitchen-card__name-row">
					<text class="kitchen-card__name">{{ currentKitchenDisplayName }}</text>
					<view v-if="currentKitchenName" class="kitchen-card__name-edit" @tap.stop="$emit('open-kitchen-name-sheet')">
						<up-icon name="edit-pen" size="15" color="#6e5f50"></up-icon>
					</view>
				</view>
				<text class="kitchen-card__meta">{{ currentKitchenMetaText }}</text>
				<view v-if="currentKitchenName" class="kitchen-card__tags">
					<view class="kitchen-card__tag kitchen-card__tag--members">
						<text class="kitchen-card__tag-value kitchen-card__tag-value--number">{{ kitchenMembersCount }}</text>
						<text class="kitchen-card__tag-label">成员</text>
					</view>
					<view v-if="currentKitchenRoleLabel" class="kitchen-card__tag kitchen-card__tag--role">
						<text class="kitchen-card__tag-value">{{ currentKitchenRoleLabel }}</text>
						<text class="kitchen-card__tag-label">身份</text>
					</view>
					<view v-if="canSwitchKitchen" class="kitchen-card__tag kitchen-card__tag--spaces">
						<text class="kitchen-card__tag-value kitchen-card__tag-value--number">{{ kitchenOptionsCount }}</text>
						<text class="kitchen-card__tag-label">空间</text>
					</view>
				</view>
			</view>

			<view class="kitchen-actions">
				<view
					class="kitchen-actions__primary"
					hover-class="kitchen-actions__primary--pressed"
					hover-start-time="0"
					hover-stay-time="180"
					hover-stop-propagation
					@tap="$emit('open-invite-sheet')"
				>
					<view class="kitchen-actions__primary-sweep"></view>
					<view class="kitchen-actions__primary-icon">
						<image
							class="kitchen-actions__primary-icon-image"
							src="/static/icons/invite-share.svg"
							mode="aspectFit"
						/>
					</view>
					<view class="kitchen-actions__primary-body">
						<text class="kitchen-actions__primary-title">邀请成员</text>
						<text class="kitchen-actions__primary-desc">{{ inviteActionDescription }}</text>
					</view>
				</view>
			</view>
		</view>

		<view class="member-panel">
			<view class="member-panel__header">
				<view class="member-panel__heading">
					<text class="member-panel__title">空间成员</text>
					<text class="member-panel__desc">与你共同维护菜单的人</text>
				</view>
				<view class="member-panel__aside">
					<text class="member-panel__meta">{{ memberPanelSummary }}</text>
					<view v-if="hasMoreKitchenMembers" class="member-panel__inline-action" @tap="$emit('show-all-members')">
						<text class="member-panel__inline-action-text">查看全部</text>
					</view>
				</view>
			</view>

			<view v-if="visibleKitchenMembers.length" class="member-list">
				<view
					v-for="member in visibleKitchenMembers"
					:key="member.userId"
					class="member-card"
					:class="{ 'member-card--self': member.isCurrentUser, 'member-card--interactive': member.isCurrentUser }"
					:hover-class="member.isCurrentUser ? 'member-card--hover' : ''"
					@tap="$emit('member-tap', member)"
				>
					<view class="member-card__avatar-wrap">
						<view class="member-card__avatar">
							<image v-if="member.avatarUrl" class="member-card__avatar-image" :src="member.avatarUrl" mode="aspectFill"></image>
							<text v-else>{{ memberInitial(member) }}</text>
						</view>
						<view v-if="member.isCurrentUser" class="member-card__avatar-badge">
							<up-icon name="checkmark-circle-fill" size="12" color="#ffffff"></up-icon>
						</view>
					</view>
					<view class="member-card__body">
						<view class="member-card__top">
							<text class="member-card__name">{{ memberDisplayName(member) }}</text>
							<view class="member-card__badges">
								<text class="member-card__badge">{{ memberRoleLabel(member.role) }}</text>
								<text v-if="member.isCurrentUser" class="member-card__badge member-card__badge--self">你</text>
							</view>
						</view>
						<view class="member-card__meta-row">
							<text class="member-card__meta">{{ memberMemberDescription(member) }}</text>
							<view v-if="member.isCurrentUser" class="member-card__action">
								<text class="member-card__action-text">修改资料</text>
								<up-icon name="arrow-right" size="12" color="#8a7d70"></up-icon>
							</view>
						</view>
					</view>
				</view>
			</view>

			<view v-else class="soft-empty soft-empty--inline member-panel__empty">
				<text class="soft-empty__text">
					{{ isLoadingKitchenMembers ? '正在获取成员信息...' : '这个空间暂时只有你，邀请好友后会显示在这里。' }}
				</text>
			</view>

			<view class="member-panel__footer">
				<view class="member-panel__join-link" @tap="$emit('open-invite-code-sheet')">
					<text class="member-panel__join-link-text">已有邀请码？去加入</text>
					<up-icon name="arrow-right" size="14" color="#7f7265"></up-icon>
				</view>
			</view>
		</view>

		<view v-if="showLeaveAction" class="kitchen-danger-zone">
			<view
				class="kitchen-danger-zone__action"
				:class="{ 'kitchen-danger-zone__action--loading': isLeavingKitchen }"
				hover-class="kitchen-danger-zone__action--pressed"
				hover-start-time="0"
				hover-stay-time="160"
				@tap="!isLeavingKitchen && $emit('leave-kitchen')"
			>
				<up-icon name="arrow-left" size="12" color="#d13d45"></up-icon>
				<text class="kitchen-danger-zone__action-text">{{ isLeavingKitchen ? '退出中...' : '退出当前空间' }}</text>
			</view>
		</view>
	</view>
</template>

<script>
const identity = (value = '') => value

export default {
	name: 'KitchenSection',
	props: {
		currentKitchenName: {
			type: String,
			default: ''
		},
		canSwitchKitchen: {
			type: Boolean,
			default: false
		},
		kitchenConnectionLabel: {
			type: String,
			default: ''
		},
		isKitchenConnected: {
			type: Boolean,
			default: false
		},
		currentKitchenDisplayName: {
			type: String,
			default: ''
		},
		currentKitchenMetaText: {
			type: String,
			default: ''
		},
		kitchenMembersCount: {
			type: Number,
			default: 0
		},
		currentKitchenRoleLabel: {
			type: String,
			default: ''
		},
		kitchenOptionsCount: {
			type: Number,
			default: 0
		},
		inviteActionDescription: {
			type: String,
			default: ''
		},
		memberPanelSummary: {
			type: String,
			default: ''
		},
		hasMoreKitchenMembers: {
			type: Boolean,
			default: false
		},
		visibleKitchenMembers: {
			type: Array,
			default: () => []
		},
		isLoadingKitchenMembers: {
			type: Boolean,
			default: false
		},
		showLeaveAction: {
			type: Boolean,
			default: false
		},
		isLeavingKitchen: {
			type: Boolean,
			default: false
		},
		memberInitial: {
			type: Function,
			default: identity
		},
		memberDisplayName: {
			type: Function,
			default: identity
		},
		memberRoleLabel: {
			type: Function,
			default: identity
		},
		memberMemberDescription: {
			type: Function,
			default: identity
		}
	},
	emits: [
		'member-tap',
		'open-invite-code-sheet',
		'open-invite-sheet',
		'open-kitchen-name-sheet',
		'open-kitchen-selector',
		'leave-kitchen',
		'show-all-members'
	]
}
</script>

<style lang="scss" scoped>
@import './kitchen-section.scss';
</style>
