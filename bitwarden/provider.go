package bitwarden

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Provider -
func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{ // TODO allow specification of 'bw config server' etc. Ensure env vars don't collide. maybe decline to accept sensitive ones this way?
			"session_key": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true, // TODO
				Sensitive:     true,
				ConflictsWith: []string{"master_password"},
			},
			"master_password": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Sensitive:    true,
				AtLeastOneOf: []string{"email", "client_secret"},
			},
			"email": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true, // TODO
			},
			"user_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"client_id": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				RequiredWith: []string{"client_secret"},
			},
			"client_secret": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Sensitive:    true,
				RequiredWith: []string{"client_id"},
			},
			"two_step_method": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  0,
			},
			"two_step_code": &schema.Schema{
				Type:      schema.TypeString, // TODO always true?
				Optional:  true,
				Sensitive: true,
			},
			"server": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "https://bitwarden.com",
			},
		},
		ResourcesMap: map[string]*schema.Resource{},
		DataSourcesMap: map[string]*schema.Resource{
			"bitwarden_item": dataSourceItem(),
		},
		ConfigureContextFunc: providerConfigure,
	}
}

func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	sessionKey := d.Get("session_key").(string)
	masterPassword := d.Get("master_password").(string)
	email := d.Get("email").(string)
	userId := d.Get("user_id").(string)
	clientId := d.Get("client_id").(string)
	clientSecret := d.Get("client_secret").(string)
	//twoStepMethod := d.Get("two_step_method").(int) // TODO
	//twoStepCode := d.Get("two_step_code").(string) // TODO
	server := d.Get("server").(string)

	var diags diag.Diagnostics

	c, err := NewClient(email, masterPassword, server, clientId, clientSecret, userId, sessionKey)
	if err != nil {
		return nil, append(diags, diag.FromErr(err)...)
	}
	return c, diags
}
