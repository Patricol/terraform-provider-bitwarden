package bitwarden

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceItem() *schema.Resource {
	// Using weird workarounds until this is resolved: https://github.com/hashicorp/terraform-plugin-sdk/issues/616
	return &schema.Resource{
		ReadContext: dataSourceItemRead,
		Schema: map[string]*schema.Schema{
			"items": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"object": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"organization_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"folder_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"type": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"notes": {
							Type:      schema.TypeString,
							Computed:  true,
							Sensitive: true,
						},
						"favorite": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"fields": {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"type": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"value": { // NOTE this is a string even when it's a bool field.
										Type:      schema.TypeString,
										Computed:  true,
										Sensitive: true,
									},
								},
							},
						},
						"secure_note": {
							Type:     schema.TypeList,
							Computed: true,
							//MinItems: 1,
							//MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"type": {
										Type:     schema.TypeInt,
										Computed: true,
									},
								},
							},
						},
						"identity": {
							Type:     schema.TypeList,
							Computed: true,
							//MinItems: 1,
							//MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"title": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"first_name": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"middle_name": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"last_name": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"address1": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"address2": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"address3": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"city": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"state": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"postal_code": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"country": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"company": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"email": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"phone": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"ssn": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"username": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"passport_number": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"license_number": {
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
						},
						"card": {
							Type:     schema.TypeList,
							Computed: true,
							//MinItems: 1,
							//MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"cardholder_name": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"brand": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"number": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"exp_month": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"exp_year": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"code": {
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
						},
						"login": {
							Type:     schema.TypeList,
							Computed: true,
							//MinItems: 1,
							//MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"uris": {
										Type:     schema.TypeList,
										Computed: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"match": {
													Type:     schema.TypeInt, // TODO sometimes null
													Computed: true,
												},
												"uri": {
													Type:     schema.TypeString,
													Computed: true,
												},
											},
										},
									},
									"username": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"password": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"totp": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"password_revision_date": {
										Type:     schema.TypeString, // TODO sometimes null
										Computed: true,
									},
								},
							},
						},
						"collection_ids": {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Schema{
								Type: schema.TypeString, // TODO unknown type/format
							},
						},
						"revision_date": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

// TODO mark sensitive pieces.

func dataSourceItemRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*Client)

	var diags diag.Diagnostics
	data, err := c.bwListItems()
	if err != nil {
		return diag.FromErr(err)
	}
	enclosure := &map[string]interface{}{
		"real": *data,
	}
	err = encloseMaps(enclosure, &map[string]interface{}{
		"real": []map[string]interface{}{
			{
				"card":        "",
				"identity":    "",
				"login":       "",
				"secure_note": "",
			},
		},
	})
	//return diag.FromErr(fmt.Errorf("%v\n\n%v", enclosure, (*enclosure)["real"]))

	if err := d.Set("items", (*enclosure)["real"]); err != nil {
		newErr := fmt.Errorf("%s\n\n%v", err.Error(), data)
		return diag.FromErr(newErr)
	}

	// always run
	d.SetId(strconv.FormatInt(time.Now().Unix(), 10)) // TODO use real ID for this.

	return diags
}
