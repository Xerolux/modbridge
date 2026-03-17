<template>
  <div class="p-4 sm:p-6 flex flex-col gap-4">
    <div class="flex flex-col sm:flex-row sm:justify-between sm:items-center gap-3">
      <h1 class="text-xl sm:text-2xl font-bold text-gray-200">User Management</h1>
      <Button @click="openCreateModal" icon="pi pi-plus" label="Add User" class="w-full sm:w-auto" />
    </div>

    <div v-if="loading" class="flex justify-center py-12">
      <i class="pi pi-spin pi-spinner text-4xl text-blue-500"></i>
    </div>

    <div v-else-if="error" class="flex flex-col items-center justify-center py-12">
      <i class="pi pi-exclamation-triangle text-4xl text-red-500 mb-4"></i>
      <p class="text-red-400">{{ error }}</p>
      <Button @click="loadUsers" label="Retry" class="mt-4" />
    </div>

    <DataTable
      v-else
      :value="users"
      :paginator="users.length > 10"
      :rows="10"
      :rowsPerPageOptions="[10, 25, 50]"
      stripedRows
      responsiveLayout="scroll"
      class="p-datatable-sm"
    >
      <Column field="username" header="Username" sortable>
        <template #body="{ data }">
          <div class="flex items-center gap-2">
            <i class="pi pi-user text-gray-400"></i>
            <span class="text-gray-200">{{ data.username }}</span>
          </div>
        </template>
      </Column>
      <Column field="email" header="Email" sortable class="hidden sm:table-cell">
        <template #body="{ data }">
          <span class="text-gray-300">{{ data.email || '-' }}</span>
        </template>
      </Column>
      <Column field="role" header="Role" sortable>
        <template #body="{ data }">
          <Tag :value="data.role" :severity="getRoleSeverity(data.role)" />
        </template>
      </Column>
      <Column field="enabled" header="Status" sortable>
        <template #body="{ data }">
          <Tag
            :value="data.enabled ? 'Enabled' : 'Disabled'"
            :severity="data.enabled ? 'success' : 'danger'"
          />
        </template>
      </Column>
      <Column header="Actions" :exportable="false">
        <template #body="{ data }">
          <div class="flex gap-2">
            <Button
              icon="pi pi-pencil"
              size="small"
              text
              @click="editUser(data)"
              v-tooltip="'Edit user'"
            />
            <Button
              icon="pi pi-trash"
              size="small"
              text
              severity="danger"
              @click="confirmDeleteUser(data)"
              v-tooltip="'Delete user'"
            />
          </div>
        </template>
      </Column>
      <template #empty>
        <div class="text-center py-8 text-gray-500">
          <i class="pi pi-users text-4xl mb-2 block"></i>
          <p>No users found</p>
        </div>
      </template>
    </DataTable>

    <Dialog
      v-model:visible="showModal"
      :header="isEditMode ? 'Edit User' : 'Create User'"
      modal
      class="w-full max-w-md mx-4"
    >
      <div class="flex flex-col gap-4">
        <div>
          <label class="block text-sm font-medium text-gray-300 mb-1">Username</label>
          <InputText v-model="formData.username" class="w-full" :disabled="isEditMode" />
        </div>
        <div>
          <label class="block text-sm font-medium text-gray-300 mb-1">Email</label>
          <InputText v-model="formData.email" type="email" class="w-full" />
        </div>
        <div>
          <label class="block text-sm font-medium text-gray-300 mb-1">Role</label>
          <Dropdown
            v-model="formData.role"
            :options="roles"
            optionLabel="label"
            optionValue="value"
            class="w-full"
          />
        </div>
        <div v-if="!isEditMode">
          <label class="block text-sm font-medium text-gray-300 mb-1">Password</label>
          <Password v-model="formData.password" :feedback="true" toggleMask class="w-full" />
        </div>
        <div class="flex items-center gap-2">
          <Checkbox v-model="formData.enabled" binary inputId="enabled-cb" />
          <label for="enabled-cb" class="text-sm text-gray-300">Enabled</label>
        </div>
      </div>
      <template #footer>
        <Button label="Cancel" severity="secondary" @click="closeModal" />
        <Button :label="isEditMode ? 'Update' : 'Create'" @click="saveUser" :loading="saving" />
      </template>
    </Dialog>

    <Toast />
    <ConfirmDialog />
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue';
import axios from '../../axios.js';
import DataTable from 'primevue/datatable';
import Column from 'primevue/column';
import Button from 'primevue/button';
import Dialog from 'primevue/dialog';
import InputText from 'primevue/inputtext';
import Dropdown from 'primevue/dropdown';
import Password from 'primevue/password';
import Checkbox from 'primevue/checkbox';
import Tag from 'primevue/tag';
import Toast from 'primevue/toast';
import ConfirmDialog from 'primevue/confirmdialog';
import { useToast } from 'primevue/usetoast';
import { useConfirm } from 'primevue/useconfirm';

const users = ref([]);
const loading = ref(true);
const error = ref(null);
const saving = ref(false);
const showModal = ref(false);
const isEditMode = ref(false);
const toast = useToast();
const confirm = useConfirm();

const formData = ref({
  id: null,
  username: '',
  email: '',
  role: 'viewer',
  password: '',
  enabled: true
});

const roles = [
  { label: 'Admin', value: 'admin' },
  { label: 'Operator', value: 'operator' },
  { label: 'Viewer', value: 'viewer' },
  { label: 'Auditor', value: 'auditor' }
];

const loadUsers = async () => {
  loading.value = true;
  error.value = null;
  try {
    const response = await axios.get('/api/users');
    users.value = response.data || [];
  } catch (e) {
    error.value = e.response?.data || 'Failed to load users';
    console.error('Failed to load users:', e);
  } finally {
    loading.value = false;
  }
};

const openCreateModal = () => {
  isEditMode.value = false;
  formData.value = {
    id: null,
    username: '',
    email: '',
    role: 'viewer',
    password: '',
    enabled: true
  };
  showModal.value = true;
};

const editUser = (user) => {
  isEditMode.value = true;
  formData.value = { ...user, password: '' };
  showModal.value = true;
};

const saveUser = async () => {
  saving.value = true;
  try {
    if (isEditMode.value) {
      const updateData = { ...formData.value };
      delete updateData.password;
      await axios.put(`/api/users/${formData.value.id}`, updateData);
      toast.add({ severity: 'success', summary: 'Success', detail: 'User updated', life: 3000 });
    } else {
      await axios.post('/api/users', formData.value);
      toast.add({ severity: 'success', summary: 'Success', detail: 'User created', life: 3000 });
    }
    closeModal();
    await loadUsers();
  } catch (e) {
    const msg = e.response?.data || 'Failed to save user';
    toast.add({ severity: 'error', summary: 'Error', detail: msg, life: 5000 });
  } finally {
    saving.value = false;
  }
};

const confirmDeleteUser = (user) => {
  confirm.require({
    message: `Are you sure you want to delete user "${user.username}"?`,
    header: 'Confirm Delete',
    icon: 'pi pi-exclamation-triangle',
    acceptLabel: 'Delete',
    rejectLabel: 'Cancel',
    accept: async () => {
      try {
        await axios.delete(`/api/users/${user.id}`);
        toast.add({ severity: 'success', summary: 'Success', detail: 'User deleted', life: 3000 });
        await loadUsers();
      } catch (e) {
        toast.add({ severity: 'error', summary: 'Error', detail: 'Failed to delete user', life: 5000 });
      }
    }
  });
};

const closeModal = () => {
  showModal.value = false;
  formData.value = {
    id: null,
    username: '',
    email: '',
    role: 'viewer',
    password: '',
    enabled: true
  };
};

const getRoleSeverity = (role) => {
  const severities = {
    admin: 'danger',
    operator: 'info',
    viewer: 'success',
    auditor: 'warn'
  };
  return severities[role] || 'secondary';
};

onMounted(() => {
  loadUsers();
});
</script>
