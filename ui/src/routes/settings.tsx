// SPDX-License-Identifier: MPL-2.0

import * as React from "react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { Helmet } from "react-helmet";
import { Button } from "../components/tremor/Button";
import { Input } from "../components/tremor/Input";
import { Label } from "../components/tremor/Label";
import { useToast } from "../hooks/useToast";
import { MenuProvider } from "../components/providers/MenuProvider";
import { MenuTrigger } from "../components/MenuTrigger";
import { RiSettings4Line } from "@remixicon/react";
import { useQueryApi } from "../hooks/useQueryApi";
import { useAuth } from "../lib/auth";
import { useEffect } from "react";

export const Route = createFileRoute("/settings")({
  component: Settings,
});

function Settings () {
  const navigate = useNavigate();
  const { toast } = useToast();
  const queryApi = useQueryApi();
  const { userName, userId, refreshUserName } = useAuth();

  useEffect(() => {
    if (!userName && !userId) {
      // Small delay to let AuthProvider initialize from localStorage
      const timer = setTimeout(() => {
        const jwt = localStorage.getItem("shaper-jwt");
        if (!jwt) {
          navigate({ to: "/", replace: true });
        }
      }, 500);
      return () => clearTimeout(timer);
    }
  }, [userName, userId, navigate]);

  const handleUpdateName = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const form = e.currentTarget;
    const formData = new FormData(form);
    const newName = formData.get("name") as string;

    try {
      await queryApi(`users/${userId}/name`, {
        method: "POST",
        body: { name: newName },
      });

      await refreshUserName();

      toast({
        title: "Success",
        description: "Name updated successfully.",
      });
    } catch (error) {
      toast({
        title: "Error",
        description: error instanceof Error ? error.message : "Failed to update name",
        variant: "error",
      });
    }
  };

  const handleUpdatePassword = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const form = e.currentTarget;
    const formData = new FormData(form);
    const currentPassword = formData.get("currentPassword") as string;
    const newPassword = formData.get("newPassword") as string;
    const confirmPassword = formData.get("confirmPassword") as string;

    if (newPassword !== confirmPassword) {
      toast({
        title: "Error",
        description: "New passwords do not match",
        variant: "error",
      });
      return;
    }

    try {
      await queryApi(`users/${userId}/password`, {
        method: "POST",
        body: {
          currentPassword,
          newPassword,
        },
      });

      toast({
        title: "Success",
        description: "Password updated successfully.",
      });

      // Clear the form
      form.reset();
    } catch (error) {
      toast({
        title: "Error",
        description: error instanceof Error ? error.message : "Failed to update password",
        variant: "error",
      });
    }
  };
  return (
    <MenuProvider isSettings>
      <Helmet>
        <title>Settings</title>
      </Helmet>

      <div className="px-3 pb-3 min-h-dvh flex flex-col">
        <div className="flex">
          <MenuTrigger className="pr-1.5 py-3 -ml-1.5" />
          <h1 className="font-semibold font-display flex-grow pb-4 pt-3.5">
            <RiSettings4Line className="size-4 inline ml-1 mr-1 -mt-1" />
            Settings
          </h1>
        </div>

        <div className="space-y-3 mt-2">
          <div className="bg-cbgs dark:bg-dbgs rounded-md shadow p-6">
            <h2 className="text-lg font-semibold mb-4">Update Profile</h2>
            <form onSubmit={handleUpdateName} className="space-y-4 max-w-md">
              <div>
                <Label htmlFor="name">Name</Label>
                <Input
                  key={userName}
                  id="name"
                  name="name"
                  type="text"
                  defaultValue={userName}
                  required
                  className="mt-1"
                />
              </div>
              <Button type="submit">Update Name</Button>
            </form>
          </div>

          <div className="bg-cbgs dark:bg-dbgs rounded-md shadow p-6">
            <h2 className="text-lg font-semibold mb-4">Change Password</h2>
            <form onSubmit={handleUpdatePassword} className="space-y-4 max-w-md">
              <div>
                <Label htmlFor="currentPassword">Current Password</Label>
                <Input
                  id="currentPassword"
                  name="currentPassword"
                  type="password"
                  required
                  className="mt-1"
                />
              </div>
              <div>
                <Label htmlFor="newPassword">New Password</Label>
                <Input
                  id="newPassword"
                  name="newPassword"
                  type="password"
                  required
                  minLength={8}
                  className="mt-1"
                />
              </div>
              <div>
                <Label htmlFor="confirmPassword">Confirm New Password</Label>
                <Input
                  id="confirmPassword"
                  name="confirmPassword"
                  type="password"
                  required
                  minLength={8}
                  className="mt-1"
                />
              </div>
              <Button type="submit">Change Password</Button>
            </form>
          </div>
        </div>
      </div>
    </MenuProvider>
  );
}
