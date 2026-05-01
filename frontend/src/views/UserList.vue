<script setup lang="ts">
import {
  OnyxPageLayout,
  OnyxDataGrid,
  type ColumnConfig,
  createFeature,
  DataGridFeatures,
  OnyxButton,
  OnyxModal,
  OnyxBottomBar,
  OnyxForm,
  OnyxInput, ColumnTypesFromFeatures,
} from "sit-onyx";
import {h, ref, watch} from "vue";
import UserActions from "../components/UserActions.vue";
import api from "../api.ts";

type User = {
  id: string;
  username: string;
  has_password: boolean;
};

let nextPageToken = "";
const pageToken = ref("");

const data = ref<User[]>([]);

const userToConfirmDelete = ref<string>();
const userModalOpen = ref(false);
const userModalMode = ref<"create" | "edit">();
const userId = ref<string>("");
const userUsername = ref("");
const userPassword = ref("");
const userUsernameModified = ref<boolean>(false);
const userPasswordModified = ref<boolean>(false);

const columns: ColumnConfig<User, Record<string, never>, ColumnTypesFromFeatures<typeof features>>[] = [
  {key: "username", label: "Username"},
  {key: "has_password", label: "Has Password?", type: "boolean", width: "max-content"},
  {key: "id", label: "Actions", type: "actions", width: "max-content"},
];

const deleteUser = async (id: string) => {
  await api.delete(`/api/users/${id}`);
  await reload();
  userToConfirmDelete.value = undefined;
}

const createOrUpdateUser = async () => {
  if (userModalMode.value === "create") {
    await api.post(`/api/users`, {
      username: userUsername.value,
      password: userPassword.value,
    });
    await reload();
    userModalOpen.value = false;
  } else {
    const fieldMask = [];
    if (userUsernameModified.value) {
      fieldMask.push("username");
    }
    if (userPasswordModified.value) {
      fieldMask.push("password")
    }

    await api.patch(`/api/users/${userId.value}?field_mask=${encodeURIComponent(fieldMask.join(','))}`, {
      username: userUsername.value,
      password: userPassword.value,
    });
    await reload();
    userModalOpen.value = false;
  }
}

watch([userModalOpen], ([isOpen]: [boolean]) => {
  if (!isOpen) {
    userUsername.value = "";
    userPassword.value = "";
  }
})

const withCreateButton = createFeature(() => ({
  name: Symbol("create button"),
  slots: {
    headline: (slotContent) => [
      h("div", { class: "users-headline" }, [
        ...slotContent(),
        h(OnyxButton, {
          label: "Create new User",
          color: "neutral",
          mode: "outline",
          onClick: () => {
            userModalMode.value = "create";
            userModalOpen.value = true;
          },
        }),
      ]),
    ],
  },
}));

const withActionColumn = createFeature(() => ({
  name: Symbol("action column"),
  typeRenderer: {
    actions: DataGridFeatures.createTypeRenderer({
      cell: {
        component: ({modelValue}) => {
          return h(UserActions, {
            id: modelValue,
            onDelete: (id) => {
              userToConfirmDelete.value = id
            },
            onEdit: (id) => {
              const user = data.value.find((u) => u.id === id);

              userId.value = id;
              userModalMode.value = "edit";
              userUsername.value = user?.username ?? "";
              userPassword.value = user?.has_password ? "placeholder" : "";
              userModalOpen.value = true;
            },
          });
        },
      },
    })
  }
}));

const withPaginationButton = createFeature(() => ({
  name: Symbol("pagination button"),
  slots: {
    // you could also define the headline here but please note that the OnyxDataGrid supports a native "headline" property.
    pagination: (slotContent) => {
      return [
        h('div', nextPageToken !== "" ? [
          h(OnyxButton, {
            label: "Load more",
            density: "compact",
            mode: "outline",
            color: "neutral",
            onClick: () => pageToken.value = nextPageToken
          }),
        ] : []),
        // using slotContent here to ensure that if multiple feature define the same slot, they will be merged
        ...slotContent(),
      ]
    },
    // you could also define pagination here but it is strongly advised to use the built-in "usePagination" feature instead
  },
}));

watch([pageToken], async () => {
  const response = await api.get<{
    users: User[],
    next_page_token: string
  }>(`/api/users?page_size=10&page_token=${nextPageToken}`);
  data.value.push(...response.data.users);
  nextPageToken = response.data.next_page_token ?? "";
}, {immediate: true});

const reload = async () => {
  nextPageToken = "";
  const response = await api.get<{
    users: User[],
    next_page_token: string
  }>(`/api/users?page_size=10`);
  data.value = response.data.users;
  nextPageToken = response.data.next_page_token ?? "";
}

const features = [withCreateButton, withActionColumn, withPaginationButton];
</script>

<template>
  <OnyxPageLayout>
    <OnyxDataGrid headline="Users" :data :columns :features></OnyxDataGrid>
    <OnyxModal :open="userToConfirmDelete !== undefined" label="Confirm Delete" :alert="true"
               @update:open="userToConfirmDelete = undefined">
      <div class="modal">
        <span class="modal__description">
          Confirm deletion of user {{ data.find((u) => u.id === userToConfirmDelete)?.username }}
        </span>
      </div>

      <template #footer>
        <OnyxBottomBar>
          <OnyxButton label="Cancel" color="neutral" mode="plain" @click="userToConfirmDelete = undefined"/>
          <OnyxButton label="Delete" color="danger" mode="plain" @click="deleteUser(userToConfirmDelete!)"/>
        </OnyxBottomBar>
      </template>
    </OnyxModal>
    <OnyxModal v-model:open="userModalOpen" :label="userModalMode === 'create' ? 'Create User' : 'Edit User'">
      <template #description>
        <template v-if="userModalMode === 'create'">This user will get full access to the Sensor and User management.
        </template>
        <template v-if="userModalMode === 'edit'">Edit existing user's credentials.</template>
      </template>
      <div class="modal">
        <OnyxForm class="form">
          <OnyxInput label="Username" v-model="userUsername" @change="userUsernameModified = true"
                     pattern="/^[a-zA-Z][a-zA-Z0-9._-]*$/" :required="true"/>
          <OnyxInput label="Password" v-model="userPassword" @change="userPasswordModified = true" type="password" />
        </OnyxForm>
      </div>
      <template #footer>
        <OnyxBottomBar>
          <OnyxButton label="Close" color="neutral" mode="plain" @click="userModalOpen = false"/>
          <OnyxButton label="Create" mode="plain" @click="createOrUpdateUser"/>
        </OnyxBottomBar>
      </template>
    </OnyxModal>
  </OnyxPageLayout>
</template>

<style scoped>
:deep(.onyx-table-wrapper td:not(:last-child)) {
  line-height: 2rem;
}

:deep(.users-headline) {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
}

.modal {
  padding: var(--onyx-density-xl) var(--onyx-modal-padding-inline);

  &__description {
    color: var(--onyx-color-text-icons-info-intense);
    white-space: pre-line;
  }
}

.form {
  display: flex;
  flex-direction: column;
  min-width: 25vw;
  gap: var(--onyx-grid-gutter);
}
</style>